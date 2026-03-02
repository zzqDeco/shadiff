package dbhook

import (
	"context"

	"shadiff/internal/model"
)

// DBHook is the interface for capturing database operations
type DBHook interface {
	// Start starts the database proxy, listening on listenAddr and forwarding to targetAddr
	Start(ctx context.Context) error
	// Stop stops the proxy
	Stop() error
	// SideEffects returns the channel of captured side effects
	SideEffects() <-chan model.SideEffect
	// Type returns the database type
	Type() string
}

// Config is the common configuration for DB proxies
type Config struct {
	DBType     string // mysql / postgres / mongo
	ListenAddr string // proxy listen address
	TargetAddr string // real DB address
}

// NewHook creates the corresponding DB hook based on the type
func NewHook(cfg Config) (DBHook, error) {
	switch cfg.DBType {
	case "mysql":
		return NewMySQLHook(cfg.ListenAddr, cfg.TargetAddr), nil
	case "postgres":
		return NewPostgresHook(cfg.ListenAddr, cfg.TargetAddr), nil
	case "mongo":
		return NewMongoHook(cfg.ListenAddr, cfg.TargetAddr), nil
	default:
		return nil, &UnsupportedDBError{DBType: cfg.DBType}
	}
}

// UnsupportedDBError represents an unsupported database type error
type UnsupportedDBError struct {
	DBType string
}

func (e *UnsupportedDBError) Error() string {
	return "unsupported database type: " + e.DBType
}
