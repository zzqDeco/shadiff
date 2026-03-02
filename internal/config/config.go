package config

// AppConfig is the root configuration.
type AppConfig struct {
	Capture CaptureConfig `json:"capture"` // Capture configuration
	Replay  ReplayConfig  `json:"replay"`  // Replay configuration
	Diff    DiffConfig    `json:"diff"`    // Diff configuration
	Storage StorageConfig `json:"storage"` // Storage configuration
	Log     LogConfig     `json:"log"`     // Log configuration
}

// CaptureConfig holds capture settings.
type CaptureConfig struct {
	ListenAddr   string        `json:"listenAddr"`   // HTTP proxy listen address (default: ":18080")
	MaxBodySize  int64         `json:"maxBodySize"`  // Max body recording size (bytes)
	ExcludePaths []string      `json:"excludePaths"` // Excluded path prefixes
	DBProxies    []DBProxyConfig `json:"dbProxies"`  // DB proxy configuration list
}

// DBProxyConfig holds database proxy settings.
type DBProxyConfig struct {
	Type       string `json:"type"`       // mysql / postgres / mongo
	ListenAddr string `json:"listenAddr"` // Proxy listen address
	TargetAddr string `json:"targetAddr"` // Actual DB address
}

// ReplayConfig holds replay settings.
type ReplayConfig struct {
	Concurrency int    `json:"concurrency"` // Concurrency level
	Timeout     string `json:"timeout"`     // Per-request timeout
	RetryCount  int    `json:"retryCount"`  // Retry count
	DelayMs     int    `json:"delayMs"`     // Delay between requests in ms
}

// DiffConfig holds diff/comparison settings.
type DiffConfig struct {
	IgnoreHeaders []string `json:"ignoreHeaders"` // Headers to ignore
	IgnoreOrder   bool     `json:"ignoreOrder"`   // Ignore JSON array order
	MaxDiffs      int      `json:"maxDiffs"`      // Max number of diffs
	Rules         []Rule   `json:"rules"`         // Built-in diff rules
	RulesFile     string   `json:"rulesFile"`     // External rules file path
}

// Rule defines a diff rule.
type Rule struct {
	Name    string   `json:"name"`    // Rule name
	Kind    string   `json:"kind"`    // ignore / transform / custom
	Paths   []string `json:"paths"`   // JSON paths to match (supports glob)
	Pattern string   `json:"pattern"` // Value regex match (optional)
	Matcher string   `json:"matcher"` // Custom matcher name (optional)
}

// StorageConfig holds storage settings.
type StorageConfig struct {
	DataDir     string `json:"dataDir"`     // Data directory (default: ~/.shadiff)
	MaxSessions int    `json:"maxSessions"` // Max number of retained sessions
}

// LogConfig holds log settings.
type LogConfig struct {
	Level  string `json:"level"`  // debug / info / warn / error
	LogDir string `json:"logDir"` // Log directory
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Capture: CaptureConfig{
			ListenAddr:  ":18080",
			MaxBodySize: 10 * 1024 * 1024, // 10MB
		},
		Replay: ReplayConfig{
			Concurrency: 1,
			Timeout:     "30s",
		},
		Diff: DiffConfig{
			MaxDiffs: 1000,
			IgnoreHeaders: []string{
				"Date", "X-Request-Id", "X-Trace-Id",
				"Server", "Content-Length",
			},
		},
		Storage: StorageConfig{
			MaxSessions: 100,
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}
