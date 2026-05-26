package model

import "shadiff/internal/dbtype"

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

	Database *DatabaseSideEffect `json:"database,omitempty"`
	HTTP     *HTTPSideEffect     `json:"http,omitempty"`
}

// DatabaseSideEffect contains database protocol-specific side-effect payloads.
type DatabaseSideEffect struct {
	Type  string           `json:"type"` // mysql / postgres / mongo / redis
	SQL   *SQLSideEffect   `json:"sql,omitempty"`
	Mongo *MongoSideEffect `json:"mongo,omitempty"`
	Redis *RedisSideEffect `json:"redis,omitempty"`
}

// SQLSideEffect contains SQL database side-effect details.
type SQLSideEffect struct {
	Query    string `json:"query,omitempty"`
	Args     []any  `json:"args,omitempty"`
	RowCount int64  `json:"rowCount,omitempty"`
}

// MongoSideEffect contains MongoDB side-effect details.
type MongoSideEffect struct {
	Database   string `json:"database,omitempty"`
	Collection string `json:"collection,omitempty"`
	Operation  string `json:"operation,omitempty"`
	Filter     any    `json:"filter,omitempty"`
	Update     any    `json:"update,omitempty"`
	Documents  any    `json:"documents,omitempty"`
	DocCount   int64  `json:"docCount,omitempty"`
}

// RedisSideEffect contains Redis side-effect details.
type RedisSideEffect struct {
	Command string   `json:"command,omitempty"`
	Key     string   `json:"key,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// HTTPSideEffect contains external HTTP call side-effect details.
type HTTPSideEffect struct {
	Request  *HTTPRequest  `json:"request,omitempty"`
	Response *HTTPResponse `json:"response,omitempty"`
}

// NewSQLSideEffect builds a database side effect for SQL protocols.
func NewSQLSideEffect(dbType, query string, timestamp int64) SideEffect {
	return SideEffect{
		Type:      SideEffectDB,
		Timestamp: timestamp,
		Database: &DatabaseSideEffect{
			Type: dbType,
			SQL:  &SQLSideEffect{Query: query},
		},
	}
}

// NewMongoSideEffect builds a MongoDB side effect.
func NewMongoSideEffect(payload MongoSideEffect, timestamp int64) SideEffect {
	return SideEffect{
		Type:      SideEffectDB,
		Timestamp: timestamp,
		Database: &DatabaseSideEffect{
			Type:  dbtype.Mongo,
			Mongo: &payload,
		},
	}
}

// NewRedisSideEffect builds a Redis side effect.
func NewRedisSideEffect(command, key string, args []string, timestamp int64) SideEffect {
	return SideEffect{
		Type:      SideEffectDB,
		Timestamp: timestamp,
		Database: &DatabaseSideEffect{
			Type: dbtype.Redis,
			Redis: &RedisSideEffect{
				Command: command,
				Key:     key,
				Args:    append([]string(nil), args...),
			},
		},
	}
}

// DatabaseType returns the database protocol type for a database side effect.
func (e SideEffect) DatabaseType() string {
	if e.Database == nil {
		return ""
	}
	return e.Database.Type
}

// SQL returns SQL payload details when present.
func (e SideEffect) SQL() *SQLSideEffect {
	if e.Database == nil {
		return nil
	}
	return e.Database.SQL
}

// Mongo returns MongoDB payload details when present.
func (e SideEffect) Mongo() *MongoSideEffect {
	if e.Database == nil {
		return nil
	}
	return e.Database.Mongo
}

// Redis returns Redis payload details when present.
func (e SideEffect) Redis() *RedisSideEffect {
	if e.Database == nil {
		return nil
	}
	return e.Database.Redis
}
