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

	if err := validateDBProxies("capture.dbProxies", cfg.Capture.DBProxies); err != nil {
		return err
	}
	if err := validateDBProxies("replay.dbProxies", cfg.Replay.DBProxies); err != nil {
		return err
	}

	return nil
}

func validateDBProxies(path string, proxies []DBProxyConfig) error {
	for i, proxy := range proxies {
		if proxy.Type != "mysql" && proxy.Type != "postgres" && proxy.Type != "mongo" {
			return fmt.Errorf("%s[%d].type must be mysql, postgres, or mongo", path, i)
		}
		if proxy.ListenAddr == "" {
			return fmt.Errorf("%s[%d].listenAddr must not be empty", path, i)
		}
		if proxy.TargetAddr == "" {
			return fmt.Errorf("%s[%d].targetAddr must not be empty", path, i)
		}
	}
	return nil
}
