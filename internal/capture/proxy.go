package capture

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"

	"github.com/google/uuid"
)

// Proxy is an HTTP reverse proxy that transparently forwards requests and captures request/response pairs
type Proxy struct {
	target       *url.URL
	proxy        *httputil.ReverseProxy
	recorder     *Recorder
	maxBodySize  int64
	excludePaths []string
	sequence     atomic.Int64
}

// ProxyOptions controls optional capture-time behavior.
type ProxyOptions struct {
	MaxBodySize         int64
	ExcludePathPrefixes []string
}

// NewProxy creates a reverse proxy instance.
func NewProxy(targetURL string, recorder *Recorder, opts ProxyOptions) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		target:       target,
		recorder:     recorder,
		maxBodySize:  opts.MaxBodySize,
		excludePaths: append([]string(nil), opts.ExcludePathPrefixes...),
	}

	p.proxy = &httputil.ReverseProxy{
		Director: p.director,
	}

	return p, nil
}

// ServeHTTP implements the http.Handler interface
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if p.shouldSkip(r.URL.Path) {
		p.proxy.ServeHTTP(w, r)
		logger.Info("capture skipped by path rule", "path", r.URL.Path)
		return
	}

	seq := int(p.sequence.Add(1))

	var reqBody *requestBodyCapture
	if r.Body != nil {
		reqBody = newRequestBodyCapture(r.Body, p.maxBodySize)
		r.Body = reqBody
	}

	reqMethod := r.Method
	reqPath := r.URL.Path
	reqQuery := r.URL.RawQuery
	reqHeaders := cloneHeaders(r.Header)
	scopeID := p.recorder.BeginRequestScope(startTime.UnixMilli())

	// Build HTTPRequest
	httpReq := model.HTTPRequest{
		Method:  reqMethod,
		Path:    reqPath,
		Query:   reqQuery,
		Headers: reqHeaders,
	}

	// Use ResponseRecorder to capture the response
	rr := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		maxBodySize:    p.maxBodySize,
	}

	p.proxy.ServeHTTP(rr, r)

	duration := time.Since(startTime).Milliseconds()
	if reqBody != nil {
		httpReq.Body = reqBody.Body()
		httpReq.BodyLen = reqBody.BodyLen()
	}

	// Build HTTPResponse
	httpResp := model.HTTPResponse{
		StatusCode: rr.statusCode,
		Headers:    cloneHeaders(rr.Header()),
		Body:       rr.body.Bytes(),
		BodyLen:    rr.bodyLen,
	}

	// Build Record and pass it to the recorder
	record := &model.Record{
		ID:          uuid.New().String()[:8],
		Sequence:    seq,
		Request:     httpReq,
		Response:    httpResp,
		SideEffects: []model.SideEffect{},
		Duration:    duration,
		RecordedAt:  time.Now().UnixMilli(),
	}

	if err := p.recorder.FinishRequestScope(scopeID, record); err != nil {
		logger.Error("record failed", err, "sequence", seq)
	}

	logger.CaptureEvent("request_captured",
		"method", reqMethod,
		"path", reqPath,
		"status", rr.statusCode,
		"duration_ms", duration,
		"sequence", seq,
	)
}

// director modifies the request target to the proxied service
func (p *Proxy) director(req *http.Request) {
	req.URL.Scheme = p.target.Scheme
	req.URL.Host = p.target.Host
	req.Host = p.target.Host
}

// responseRecorder is a ResponseWriter wrapper that captures response content
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        bytes.Buffer
	bodyLen     int64
	maxBodySize int64
	wroteHeader bool
}

func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.wroteHeader {
		rr.statusCode = code
		rr.wroteHeader = true
		rr.ResponseWriter.WriteHeader(code)
	}
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.bodyLen += int64(len(b))
	if rr.maxBodySize > 0 {
		remaining := rr.maxBodySize - int64(rr.body.Len())
		if remaining > 0 {
			if remaining > int64(len(b)) {
				remaining = int64(len(b))
			}
			rr.body.Write(b[:remaining])
		}
	}
	return rr.ResponseWriter.Write(b)
}

// cloneHeaders deep-copies HTTP headers
func cloneHeaders(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}
	result := make(map[string][]string, len(h))
	for k, v := range h {
		cp := make([]string, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

func (p *Proxy) shouldSkip(path string) bool {
	for _, prefix := range p.excludePaths {
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

type requestBodyCapture struct {
	body        io.ReadCloser
	captured    bytes.Buffer
	bodyLen     int64
	maxBodySize int64
}

func newRequestBodyCapture(body io.ReadCloser, maxBodySize int64) *requestBodyCapture {
	return &requestBodyCapture{
		body:        body,
		maxBodySize: maxBodySize,
	}
}

func (c *requestBodyCapture) Read(p []byte) (int, error) {
	n, err := c.body.Read(p)
	if n > 0 {
		c.bodyLen += int64(n)
		c.capture(p[:n])
	}
	return n, err
}

func (c *requestBodyCapture) Close() error {
	return c.body.Close()
}

func (c *requestBodyCapture) Body() []byte {
	return append([]byte(nil), c.captured.Bytes()...)
}

func (c *requestBodyCapture) BodyLen() int64 {
	return c.bodyLen
}

func (c *requestBodyCapture) capture(chunk []byte) {
	if c.maxBodySize == 0 {
		return
	}
	if c.maxBodySize < 0 {
		_, _ = c.captured.Write(chunk)
		return
	}

	remaining := c.maxBodySize - int64(c.captured.Len())
	if remaining <= 0 {
		return
	}
	if remaining > int64(len(chunk)) {
		remaining = int64(len(chunk))
	}
	_, _ = c.captured.Write(chunk[:remaining])
}
