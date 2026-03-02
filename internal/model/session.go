package model

// Session 表示一次完整的录制会话，包含一组相关的 API 调用记录
type Session struct {
	ID          string            `json:"id"`          // UUID 短 ID (8 字符)
	Name        string            `json:"name"`        // 用户自定义名称
	Description string            `json:"description"` // 会话描述
	Source      EndpointConfig    `json:"source"`      // 源端配置 (被录制的服务)
	Target      EndpointConfig    `json:"target"`      // 目标端配置 (回放目标)
	Tags        []string          `json:"tags"`        // 标签，用于过滤
	RecordCount int               `json:"recordCount"` // 记录数量
	CreatedAt   int64             `json:"createdAt"`   // 创建时间 (Unix ms)
	UpdatedAt   int64             `json:"updatedAt"`   // 更新时间 (Unix ms)
	Status      SessionStatus     `json:"status"`      // recording / completed / replayed
	Metadata    map[string]string `json:"metadata"`    // 扩展元数据
}

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionRecording SessionStatus = "recording"
	SessionCompleted SessionStatus = "completed"
	SessionReplayed  SessionStatus = "replayed"
)

// EndpointConfig 端点配置
type EndpointConfig struct {
	BaseURL string            `json:"baseURL"` // e.g. "http://localhost:8080"
	Headers map[string]string `json:"headers"` // 默认附加头
}

// SessionFilter 会话过滤条件
type SessionFilter struct {
	Name   string   `json:"name,omitempty"`   // 名称模糊匹配
	Status string   `json:"status,omitempty"` // 状态过滤
	Tags   []string `json:"tags,omitempty"`   // 标签过滤
}
