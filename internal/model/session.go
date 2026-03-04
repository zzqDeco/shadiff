package model

// Session represents a complete recording session containing a set of related API call records
type Session struct {
	ID          string            `json:"id"`                    // UUID short ID (8 characters)
	Name        string            `json:"name"`                  // User-defined name
	Description string            `json:"description"`           // Session description
	Source      EndpointConfig    `json:"source"`                // Source endpoint config (the service being recorded)
	Target      EndpointConfig    `json:"target"`                // Target endpoint config (replay target)
	Tags        []string          `json:"tags"`                  // Tags for filtering
	RecordCount int               `json:"recordCount"`           // Number of records
	CreatedAt   int64             `json:"createdAt"`             // Creation time (Unix ms)
	UpdatedAt   int64             `json:"updatedAt"`             // Update time (Unix ms)
	Status      SessionStatus     `json:"status"`                // recording / completed / replayed
	Metadata    map[string]string `json:"metadata"`              // Extended metadata
	PID         int               `json:"pid,omitempty"`         // Record process PID (set in daemon mode)
	DaemonMode  bool              `json:"daemonMode,omitempty"`  // Whether running as daemon
}

// SessionStatus represents the session status
type SessionStatus string

const (
	SessionRecording SessionStatus = "recording"
	SessionCompleted SessionStatus = "completed"
	SessionReplayed  SessionStatus = "replayed"
)

// EndpointConfig represents an endpoint configuration
type EndpointConfig struct {
	BaseURL string            `json:"baseURL"` // e.g. "http://localhost:8080"
	Headers map[string]string `json:"headers"` // Default additional headers
}

// SessionFilter represents session filter criteria
type SessionFilter struct {
	Name   string   `json:"name,omitempty"`   // Fuzzy match on name
	Status string   `json:"status,omitempty"` // Filter by status
	Tags   []string `json:"tags,omitempty"`   // Filter by tags
}
