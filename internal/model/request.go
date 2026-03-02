package model

// HTTPRequest HTTP 请求模型
type HTTPRequest struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`    // 不含 host 的路径
	Query   string              `json:"query"`   // 原始 query string
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
	BodyLen int64               `json:"bodyLen"` // body 长度，用于大 body 截断场景
}

// HTTPResponse HTTP 响应模型
type HTTPResponse struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	BodyLen    int64               `json:"bodyLen"`
}
