package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate checks whether the config contains supported and internally
// consistent values.
func Validate(cfg *AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Capture.MaxBodySize < 0 {
		return fmt.Errorf("capture.maxBodySize must be >= 0")
	}
	if cfg.Replay.Concurrency < 1 {
		return fmt.Errorf("replay.concurrency must be >= 1")
	}
	if cfg.Replay.DelayMs < 0 {
		return fmt.Errorf("replay.delayMs must be >= 0")
	}
	if cfg.Replay.RetryCount < 0 {
		return fmt.Errorf("replay.retryCount must be >= 0")
	}
	if _, err := time.ParseDuration(cfg.Replay.Timeout); err != nil {
		return fmt.Errorf("replay.timeout is invalid: %w", err)
	}
	if cfg.Diff.MaxDiffs < 1 {
		return fmt.Errorf("diff.maxDiffs must be >= 1")
	}
	switch strings.ToLower(cfg.Log.Level) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log.level must be one of debug, info, warn, error")
	}

	for i, proxy := range cfg.Capture.DBProxies {
		if proxy.Type != "mysql" && proxy.Type != "postgres" && proxy.Type != "mongo" {
			return fmt.Errorf("capture.dbProxies[%d].type must be mysql, postgres, or mongo", i)
		}
		if proxy.ListenAddr == "" {
			return fmt.Errorf("capture.dbProxies[%d].listenAddr must not be empty", i)
		}
		if proxy.TargetAddr == "" {
			return fmt.Errorf("capture.dbProxies[%d].targetAddr must not be empty", i)
		}
	}

	return nil
}
