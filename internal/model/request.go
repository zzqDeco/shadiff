package model

// HTTPRequest represents an HTTP request model
type HTTPRequest struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`    // Path without host
	Query   string              `json:"query"`   // Raw query string
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
	BodyLen int64               `json:"bodyLen"` // Body length, used for large body truncation scenarios
}

// HTTPResponse represents an HTTP response model
type HTTPResponse struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	BodyLen    int64               `json:"bodyLen"`
}
