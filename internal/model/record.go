package model

// Record represents the complete behavior of a single API call, including input, output, and side effects
type Record struct {
	ID          string       `json:"id"`          // Unique record ID
	SessionID   string       `json:"sessionID"`   // Owning session
	Sequence    int          `json:"sequence"`     // Sequence number within session, used for pairing
	Request     HTTPRequest  `json:"request"`      // HTTP request
	Response    HTTPResponse `json:"response"`     // HTTP response
	SideEffects []SideEffect `json:"sideEffects"`  // List of side effects
	Duration    int64        `json:"duration"`     // Request duration (ms)
	RecordedAt  int64        `json:"recordedAt"`   // Recording time (Unix ms)
	Error       string       `json:"error,omitempty"` // Collection error message
}
