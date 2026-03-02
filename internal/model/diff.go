package model

// DiffResult represents the comparison result of a pair of recorded/replayed records
type DiffResult struct {
	RecordID    string       `json:"recordID"`    // Corresponding Record ID
	Sequence    int          `json:"sequence"`    // Sequence number
	Request     HTTPRequest  `json:"request"`     // Original request (for context)
	Match       bool         `json:"match"`       // Whether it fully matches
	Differences []Difference `json:"differences"` // List of differences
}

// DifferenceKind represents the category of a difference
type DifferenceKind string

const (
	DiffStatusCode   DifferenceKind = "status_code"
	DiffHeader       DifferenceKind = "header"
	DiffBody         DifferenceKind = "body"
	DiffBodyField    DifferenceKind = "body_field"     // JSON field-level difference
	DiffDBQuery      DifferenceKind = "db_query"       // Database query difference
	DiffDBQueryCount DifferenceKind = "db_query_count" // Query count difference
	DiffMongoOp      DifferenceKind = "mongo_op"       // MongoDB operation difference
	DiffExternalCall DifferenceKind = "external_call"  // External call difference
)

// Severity represents the severity level of a difference
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Difference represents a single difference entry
type Difference struct {
	Kind     DifferenceKind `json:"kind"`     // Difference category
	Path     string         `json:"path"`     // Difference path (e.g. "body.data.items[0].name")
	Expected any            `json:"expected"` // Recorded value
	Actual   any            `json:"actual"`   // Replayed value
	Message  string         `json:"message"`  // Human-readable description
	Severity Severity       `json:"severity"` // error / warning / info
	Ignored  bool           `json:"ignored"`  // Whether ignored by a rule
	Rule     string         `json:"rule"`     // Name of the matched ignore rule
}

// DiffSummary represents the comparison summary statistics
type DiffSummary struct {
	SessionID   string `json:"sessionID"`
	TotalCount  int    `json:"totalCount"`  // Total record count
	MatchCount  int    `json:"matchCount"`  // Match count
	DiffCount   int    `json:"diffCount"`   // Difference count
	ErrorCount  int    `json:"errorCount"`  // Error-level difference count
	IgnoreCount int    `json:"ignoreCount"` // Ignored difference count
	MatchRate   float64 `json:"matchRate"`  // Match rate (0-1)
}
