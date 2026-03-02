package model

// SideEffectType represents the type of side effect
type SideEffectType string

const (
	SideEffectDB   SideEffectType = "database"  // Database operation
	SideEffectHTTP SideEffectType = "http_call" // External HTTP call
)

// SideEffect represents a side effect produced during API processing
type SideEffect struct {
	Type      SideEffectType `json:"type"`      // "database" / "http_call"
	Timestamp int64          `json:"timestamp"` // Occurrence time (Unix ms)
	Duration  int64          `json:"duration"`  // Execution duration (ms)

	// General DB fields
	DBType string `json:"dbType,omitempty"` // mysql / postgres / mongo

	// SQL databases (MySQL, PostgreSQL)
	Query    string `json:"query,omitempty"`    // SQL statement
	Args     []any  `json:"args,omitempty"`     // SQL parameters
	RowCount int64  `json:"rowCount,omitempty"` // Affected/returned row count

	// MongoDB-specific fields
	Database   string `json:"database,omitempty"`   // Database name
	Collection string `json:"collection,omitempty"` // Collection name
	Operation  string `json:"operation,omitempty"`  // find / insert / update / delete / aggregate
	Filter     any    `json:"filter,omitempty"`     // Query filter (BSON -> JSON)
	Update     any    `json:"update,omitempty"`     // Update operation
	Documents  any    `json:"documents,omitempty"`  // Inserted documents
	DocCount   int64  `json:"docCount,omitempty"`   // Affected/returned document count

	// External HTTP call fields
	HTTPReq  *HTTPRequest  `json:"httpReq,omitempty"`
	HTTPResp *HTTPResponse `json:"httpResp,omitempty"`
}
