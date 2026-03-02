package storage

import "shadiff/internal/model"

// SessionStore defines the interface for session storage
type SessionStore interface {
	Create(session *model.Session) error
	Get(id string) (*model.Session, error)
	List(filter *model.SessionFilter) ([]model.Session, error)
	Update(session *model.Session) error
	Delete(id string) error
}

// RecordStore defines the interface for record storage
type RecordStore interface {
	// AppendRecord appends a record to the session (JSONL streaming write)
	AppendRecord(sessionID string, record *model.Record) error
	// ListRecords reads all records for a session
	ListRecords(sessionID string) ([]model.Record, error)
	// GetRecord retrieves a single record
	GetRecord(sessionID string, recordID string) (*model.Record, error)
	// CountRecords returns the number of records in a session
	CountRecords(sessionID string) (int, error)
}

// DiffStore defines the interface for diff result storage
type DiffStore interface {
	// SaveResults saves diff results
	SaveResults(sessionID string, results []model.DiffResult) error
	// LoadResults loads diff results
	LoadResults(sessionID string) ([]model.DiffResult, error)
}
