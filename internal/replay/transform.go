package replay

import (
	"net/http"
	"strings"

	"shadiff/internal/model"
)

// TransformConfig holds the request transformation configuration
type TransformConfig struct {
	TargetBaseURL  string            // target service URL
	HeaderOverride map[string]string // headers to override
	HeaderRemove   []string          // headers to remove
}

// Transform transforms a recorded request to adapt it for the replay target
func Transform(req model.HTTPRequest, cfg TransformConfig) *http.Request {
	// Build full URL
	urlStr := cfg.TargetBaseURL + req.Path
	if req.Query != "" {
		urlStr += "?" + req.Query
	}

	httpReq, err := http.NewRequest(req.Method, urlStr, strings.NewReader(string(req.Body)))
	if err != nil {
		return nil
	}

	// Copy original headers
	for k, vs := range req.Headers {
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}

	// Remove specified headers
	for _, h := range cfg.HeaderRemove {
		httpReq.Header.Del(h)
	}

	// Override specified headers
	for k, v := range cfg.HeaderOverride {
		httpReq.Header.Set(k, v)
	}

	// Remove proxy-related headers
	httpReq.Header.Del("X-Forwarded-For")
	httpReq.Header.Del("X-Forwarded-Host")
	httpReq.Header.Del("X-Forwarded-Proto")

	return httpReq
}
