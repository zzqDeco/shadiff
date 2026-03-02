package capture

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"

	"github.com/google/uuid"
)

// Proxy is an HTTP reverse proxy that transparently forwards requests and captures request/response pairs
type Proxy struct {
	target   *url.URL
	proxy    *httputil.ReverseProxy
	recorder *Recorder
	sequence atomic.Int64
}

// NewProxy creates a reverse proxy instance
func NewProxy(targetURL string, recorder *Recorder) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		target:   target,
		recorder: recorder,
	}

	p.proxy = &httputil.ReverseProxy{
		Director: p.director,
	}

	return p, nil
}

// ServeHTTP implements the http.Handler interface
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	seq := int(p.sequence.Add(1))

	// Read request body
	var reqBody []byte
	if r.Body != nil {
		reqBody, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	// Build HTTPRequest
	httpReq := model.HTTPRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.RawQuery,
		Headers: cloneHeaders(r.Header),
		Body:    reqBody,
		BodyLen: int64(len(reqBody)),
	}

	// Use ResponseRecorder to capture the response
	rr := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	p.proxy.ServeHTTP(rr, r)

	duration := time.Since(startTime).Milliseconds()

	// Build HTTPResponse
	httpResp := model.HTTPResponse{
		StatusCode: rr.statusCode,
		Headers:    cloneHeaders(rr.Header()),
		Body:       rr.body.Bytes(),
		BodyLen:    int64(rr.body.Len()),
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

	if err := p.recorder.Record(record); err != nil {
		logger.Error("record failed", err, "sequence", seq)
	}

	logger.CaptureEvent("request_captured",
		"method", r.Method,
		"path", r.URL.Path,
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
	statusCode int
	body       bytes.Buffer
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
	rr.body.Write(b)
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
