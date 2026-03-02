package config

// AppConfig 根配置
type AppConfig struct {
	Capture CaptureConfig `json:"capture"` // 采集配置
	Replay  ReplayConfig  `json:"replay"`  // 回放配置
	Diff    DiffConfig    `json:"diff"`    // 对拍配置
	Storage StorageConfig `json:"storage"` // 存储配置
	Log     LogConfig     `json:"log"`     // 日志配置
}

// CaptureConfig 采集配置
type CaptureConfig struct {
	ListenAddr   string        `json:"listenAddr"`   // HTTP 代理监听地址 (default: ":18080")
	MaxBodySize  int64         `json:"maxBodySize"`  // 最大 body 录制大小 (bytes)
	ExcludePaths []string      `json:"excludePaths"` // 排除的路径前缀
	DBProxies    []DBProxyConfig `json:"dbProxies"`  // DB 代理配置列表
}

// DBProxyConfig 数据库代理配置
type DBProxyConfig struct {
	Type       string `json:"type"`       // mysql / postgres / mongo
	ListenAddr string `json:"listenAddr"` // 代理监听地址
	TargetAddr string `json:"targetAddr"` // 真实 DB 地址
}

// ReplayConfig 回放配置
type ReplayConfig struct {
	Concurrency int    `json:"concurrency"` // 并发数
	Timeout     string `json:"timeout"`     // 单请求超时
	RetryCount  int    `json:"retryCount"`  // 重试次数
	DelayMs     int    `json:"delayMs"`     // 请求间延迟 ms
}

// DiffConfig 对拍配置
type DiffConfig struct {
	IgnoreHeaders []string `json:"ignoreHeaders"` // 忽略的 header 列表
	IgnoreOrder   bool     `json:"ignoreOrder"`   // 忽略 JSON 数组顺序
	MaxDiffs      int      `json:"maxDiffs"`      // 最大差异数
	Rules         []Rule   `json:"rules"`         // 内置对拍规则
	RulesFile     string   `json:"rulesFile"`     // 外部规则文件路径
}

// Rule 对拍规则
type Rule struct {
	Name    string   `json:"name"`    // 规则名
	Kind    string   `json:"kind"`    // ignore / transform / custom
	Paths   []string `json:"paths"`   // 匹配的 JSON 路径 (支持 glob)
	Pattern string   `json:"pattern"` // 值正则匹配 (可选)
	Matcher string   `json:"matcher"` // 自定义匹配器名 (可选)
}

// StorageConfig 存储配置
type StorageConfig struct {
	DataDir     string `json:"dataDir"`     // 数据目录 (default: ~/.shadiff)
	MaxSessions int    `json:"maxSessions"` // 最大保留会话数
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `json:"level"`  // debug / info / warn / error
	LogDir string `json:"logDir"` // 日志目录
}

// DefaultConfig 返回默认配置
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
