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

// Proxy HTTP 反向代理，透明转发请求并捕获请求/响应
type Proxy struct {
	target   *url.URL
	proxy    *httputil.ReverseProxy
	recorder *Recorder
	sequence atomic.Int64
}

// NewProxy 创建反向代理实例
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

// ServeHTTP 实现 http.Handler 接口
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	seq := int(p.sequence.Add(1))

	// 读取请求 body
	var reqBody []byte
	if r.Body != nil {
		reqBody, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	// 构造 HTTPRequest
	httpReq := model.HTTPRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.RawQuery,
		Headers: cloneHeaders(r.Header),
		Body:    reqBody,
		BodyLen: int64(len(reqBody)),
	}

	// 使用 ResponseRecorder 捕获响应
	rr := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	p.proxy.ServeHTTP(rr, r)

	duration := time.Since(startTime).Milliseconds()

	// 构造 HTTPResponse
	httpResp := model.HTTPResponse{
		StatusCode: rr.statusCode,
		Headers:    cloneHeaders(rr.Header()),
		Body:       rr.body.Bytes(),
		BodyLen:    int64(rr.body.Len()),
	}

	// 构造 Record 并交给 recorder
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

// director 修改请求目标为被代理的服务
func (p *Proxy) director(req *http.Request) {
	req.URL.Scheme = p.target.Scheme
	req.URL.Host = p.target.Host
	req.Host = p.target.Host
}

// responseRecorder 捕获响应内容的 ResponseWriter 包装器
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

// cloneHeaders 深拷贝 HTTP headers
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
