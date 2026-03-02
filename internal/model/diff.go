package model

// DiffResult 表示一对录制/回放记录的对拍结果
type DiffResult struct {
	RecordID    string       `json:"recordID"`    // 对应的 Record ID
	Sequence    int          `json:"sequence"`    // 序号
	Request     HTTPRequest  `json:"request"`     // 原始请求 (用于上下文)
	Match       bool         `json:"match"`       // 是否完全匹配
	Differences []Difference `json:"differences"` // 差异列表
}

// DifferenceKind 差异类别
type DifferenceKind string

const (
	DiffStatusCode   DifferenceKind = "status_code"
	DiffHeader       DifferenceKind = "header"
	DiffBody         DifferenceKind = "body"
	DiffBodyField    DifferenceKind = "body_field"     // JSON 字段级差异
	DiffDBQuery      DifferenceKind = "db_query"       // 数据库查询差异
	DiffDBQueryCount DifferenceKind = "db_query_count" // 查询次数差异
	DiffMongoOp      DifferenceKind = "mongo_op"       // MongoDB 操作差异
	DiffExternalCall DifferenceKind = "external_call"  // 外部调用差异
)

// Severity 差异严重级别
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Difference 单条差异
type Difference struct {
	Kind     DifferenceKind `json:"kind"`     // 差异类别
	Path     string         `json:"path"`     // 差异路径 (e.g. "body.data.items[0].name")
	Expected any            `json:"expected"` // 录制值
	Actual   any            `json:"actual"`   // 回放值
	Message  string         `json:"message"`  // 人类可读描述
	Severity Severity       `json:"severity"` // error / warning / info
	Ignored  bool           `json:"ignored"`  // 是否被规则忽略
	Rule     string         `json:"rule"`     // 命中的忽略规则名
}

// DiffSummary 对拍汇总统计
type DiffSummary struct {
	SessionID   string `json:"sessionID"`
	TotalCount  int    `json:"totalCount"`  // 总记录数
	MatchCount  int    `json:"matchCount"`  // 匹配数
	DiffCount   int    `json:"diffCount"`   // 差异数
	ErrorCount  int    `json:"errorCount"`  // 错误级差异数
	IgnoreCount int    `json:"ignoreCount"` // 被忽略的差异数
	MatchRate   float64 `json:"matchRate"`  // 匹配率 (0-1)
}
