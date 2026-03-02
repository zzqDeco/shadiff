package replay

import (
	"net/http"
	"strings"

	"shadiff/internal/model"
)

// TransformConfig 请求变换配置
type TransformConfig struct {
	TargetBaseURL  string            // 目标服务地址
	HeaderOverride map[string]string // 覆盖的请求头
	HeaderRemove   []string          // 移除的请求头
}

// Transform 对录制的请求进行变换，适配回放目标
func Transform(req model.HTTPRequest, cfg TransformConfig) *http.Request {
	// 构造完整 URL
	urlStr := cfg.TargetBaseURL + req.Path
	if req.Query != "" {
		urlStr += "?" + req.Query
	}

	httpReq, err := http.NewRequest(req.Method, urlStr, strings.NewReader(string(req.Body)))
	if err != nil {
		return nil
	}

	// 复制原始 headers
	for k, vs := range req.Headers {
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}

	// 移除指定 headers
	for _, h := range cfg.HeaderRemove {
		httpReq.Header.Del(h)
	}

	// 覆盖指定 headers
	for k, v := range cfg.HeaderOverride {
		httpReq.Header.Set(k, v)
	}

	// 移除代理相关 headers
	httpReq.Header.Del("X-Forwarded-For")
	httpReq.Header.Del("X-Forwarded-Host")
	httpReq.Header.Del("X-Forwarded-Proto")

	return httpReq
}
