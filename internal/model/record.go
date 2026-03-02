package model

// Record 表示单次 API 调用的完整行为，包括输入、输出和副作用
type Record struct {
	ID          string       `json:"id"`          // 记录唯一 ID
	SessionID   string       `json:"sessionID"`   // 所属会话
	Sequence    int          `json:"sequence"`     // 会话内序号，用于配对
	Request     HTTPRequest  `json:"request"`      // HTTP 请求
	Response    HTTPResponse `json:"response"`     // HTTP 响应
	SideEffects []SideEffect `json:"sideEffects"`  // 副作用列表
	Duration    int64        `json:"duration"`     // 请求耗时 (ms)
	RecordedAt  int64        `json:"recordedAt"`   // 录制时间 (Unix ms)
	Error       string       `json:"error,omitempty"` // 采集错误信息
}
