package model

// SideEffectType 副作用类型
type SideEffectType string

const (
	SideEffectDB   SideEffectType = "database"  // 数据库操作
	SideEffectHTTP SideEffectType = "http_call" // 外部 HTTP 调用
)

// SideEffect 表示 API 处理过程中产生的副作用
type SideEffect struct {
	Type      SideEffectType `json:"type"`      // "database" / "http_call"
	Timestamp int64          `json:"timestamp"` // 发生时间 (Unix ms)
	Duration  int64          `json:"duration"`  // 执行耗时 (ms)

	// 通用 DB 字段
	DBType string `json:"dbType,omitempty"` // mysql / postgres / mongo

	// SQL 类数据库 (MySQL, PostgreSQL)
	Query    string `json:"query,omitempty"`    // SQL 语句
	Args     []any  `json:"args,omitempty"`     // SQL 参数
	RowCount int64  `json:"rowCount,omitempty"` // 影响/返回行数

	// MongoDB 专用字段
	Database   string `json:"database,omitempty"`   // 数据库名
	Collection string `json:"collection,omitempty"` // 集合名
	Operation  string `json:"operation,omitempty"`  // find / insert / update / delete / aggregate
	Filter     any    `json:"filter,omitempty"`     // 查询条件 (BSON -> JSON)
	Update     any    `json:"update,omitempty"`     // 更新操作
	Documents  any    `json:"documents,omitempty"`  // 插入的文档
	DocCount   int64  `json:"docCount,omitempty"`   // 影响/返回文档数

	// HTTP 外部调用字段
	HTTPReq  *HTTPRequest  `json:"httpReq,omitempty"`
	HTTPResp *HTTPResponse `json:"httpResp,omitempty"`
}
