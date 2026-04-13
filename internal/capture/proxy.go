package capture

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"

	"github.com/google/uuid"
)

const requestBodySpillThreshold int64 = 1 << 20

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
	recordID := uuid.New().String()[:8]

	reqMethod := r.Method
	reqPath := r.URL.Path
	reqQuery := r.URL.RawQuery
	reqHeaders := cloneHeaders(r.Header)

	var (
		reqBody    []byte
		reqBodyLen int64
		reqBodyRef string
	)
	if r.Body != nil {
		snapshot, err := captureRequestBodySnapshot(r.Body, p.maxBodySize)
		if err != nil {
			logger.Error("read request body failed", err, "method", reqMethod, "path", reqPath)
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		defer snapshot.Cleanup()

		restoreBody, err := snapshot.Reader()
		if err != nil {
			logger.Error("restore request body failed", err, "method", reqMethod, "path", reqPath)
			http.Error(w, "failed to proxy request body", http.StatusInternalServerError)
			return
		}
		r.Body = restoreBody
		reqBody = snapshot.Body()
		reqBodyLen = snapshot.BodyLen()

		if snapshot.NeedsArtifact() {
			artifactSource, err := snapshot.ArtifactSource()
			if err != nil {
				logger.Error("open request body artifact source failed", err, "record_id", recordID)
			} else {
				reqBodyRef, err = p.recorder.SaveRequestBodyArtifact(recordID, artifactSource)
				_ = artifactSource.Close()
				if err != nil {
					logger.Error("save request body artifact failed", err, "record_id", recordID)
				}
			}
		}
	}

	scopeID := p.recorder.BeginRequestScope(startTime.UnixMilli())

	// Build HTTPRequest
	httpReq := model.HTTPRequest{
		Method:  reqMethod,
		Path:    reqPath,
		Query:   reqQuery,
		Headers: reqHeaders,
		Body:    reqBody,
		BodyRef: reqBodyRef,
		BodyLen: reqBodyLen,
	}

	// Use ResponseRecorder to capture the response
	rr := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		maxBodySize:    p.maxBodySize,
	}

	p.proxy.ServeHTTP(rr, r)

	duration := time.Since(startTime).Milliseconds()

	// Build HTTPResponse
	httpResp := model.HTTPResponse{
		StatusCode: rr.statusCode,
		Headers:    cloneHeaders(rr.Header()),
		Body:       rr.body.Bytes(),
		BodyLen:    rr.bodyLen,
	}

	// Build Record and pass it to the recorder
	record := &model.Record{
		ID:          recordID,
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
	preview        bytes.Buffer
	buffer         bytes.Buffer
	tempFile       *os.File
	tempPath       string
	bodyLen        int64
	maxPreviewSize int64
}

func captureRequestBodySnapshot(body io.ReadCloser, maxPreviewSize int64) (*requestBodyCapture, error) {
	snapshot := &requestBodyCapture{
		maxPreviewSize: maxPreviewSize,
	}
	if body == nil {
		return snapshot, nil
	}
	defer body.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if writeErr := snapshot.writeChunk(buf[:n]); writeErr != nil {
				snapshot.Cleanup()
				return nil, writeErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			snapshot.Cleanup()
			return nil, err
		}
	}

	if err := snapshot.closeTempFile(); err != nil {
		snapshot.Cleanup()
		return nil, err
	}

	return snapshot, nil
}

func (c *requestBodyCapture) Body() []byte {
	return append([]byte(nil), c.preview.Bytes()...)
}

func (c *requestBodyCapture) BodyLen() int64 {
	return c.bodyLen
}

func (c *requestBodyCapture) NeedsArtifact() bool {
	return c.bodyLen > int64(len(c.preview.Bytes()))
}

func (c *requestBodyCapture) Reader() (io.ReadCloser, error) {
	if c.tempPath != "" {
		return os.Open(c.tempPath)
	}
	return io.NopCloser(bytes.NewReader(c.buffer.Bytes())), nil
}

func (c *requestBodyCapture) ArtifactSource() (io.ReadCloser, error) {
	return c.Reader()
}

func (c *requestBodyCapture) Cleanup() {
	if c.tempFile != nil {
		_ = c.tempFile.Close()
		c.tempFile = nil
	}
	if c.tempPath != "" {
		_ = os.Remove(c.tempPath)
		c.tempPath = ""
	}
}

func (c *requestBodyCapture) capture(chunk []byte) {
	if c.maxPreviewSize == 0 {
		return
	}

	remaining := c.maxPreviewSize - int64(c.preview.Len())
	if remaining <= 0 {
		return
	}
	if remaining > int64(len(chunk)) {
		remaining = int64(len(chunk))
	}
	_, _ = c.preview.Write(chunk[:remaining])
}

func (c *requestBodyCapture) writeChunk(chunk []byte) error {
	c.bodyLen += int64(len(chunk))
	c.capture(chunk)

	if c.tempFile != nil {
		_, err := c.tempFile.Write(chunk)
		return err
	}

	if int64(c.buffer.Len()+len(chunk)) <= requestBodySpillThreshold {
		_, _ = c.buffer.Write(chunk)
		return nil
	}

	tempFile, err := os.CreateTemp("", "shadiff-request-body-*")
	if err != nil {
		return err
	}
	if _, err := tempFile.Write(c.buffer.Bytes()); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return err
	}
	c.buffer.Reset()

	if _, err := tempFile.Write(chunk); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return err
	}

	c.tempFile = tempFile
	return nil
}

func (c *requestBodyCapture) closeTempFile() error {
	if c.tempFile == nil {
		return nil
	}

	c.tempPath = c.tempFile.Name()
	if err := c.tempFile.Close(); err != nil {
		_ = os.Remove(c.tempPath)
		c.tempPath = ""
		c.tempFile = nil
		return err
	}
	c.tempFile = nil
	return nil
}
