package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStoreWithPath_CreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")

	store, err := NewStoreWithPath(path)
	if err != nil {
		t.Fatalf("NewStoreWithPath() error: %v", err)
	}

	if store.path != path {
		t.Fatalf("store path = %q, want %q", store.path, path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}
}

func TestNewStoreWithPath_InvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	if _, err := NewStoreWithPath(path); err == nil {
		t.Fatal("expected invalid config to return an error")
	}
}

func TestValidate_InvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AppConfig)
	}{
		{
			name: "negative body size",
			mutate: func(cfg *AppConfig) {
				cfg.Capture.MaxBodySize = -1
			},
		},
		{
			name: "bad timeout",
			mutate: func(cfg *AppConfig) {
				cfg.Replay.Timeout = "not-a-duration"
			},
		},
		{
			name: "bad log level",
			mutate: func(cfg *AppConfig) {
				cfg.Log.Level = "trace"
			},
		},
		{
			name: "bad db type",
			mutate: func(cfg *AppConfig) {
				cfg.Capture.DBProxies = []DBProxyConfig{{Type: "redis", ListenAddr: ":1", TargetAddr: ":2"}}
			},
		},
		{
			name: "bad replay db type",
			mutate: func(cfg *AppConfig) {
				cfg.Replay.DBProxies = []DBProxyConfig{{Type: "redis", ListenAddr: ":1", TargetAddr: ":2"}}
			},
		},
		{
			name: "empty replay db listen addr",
			mutate: func(cfg *AppConfig) {
				cfg.Replay.DBProxies = []DBProxyConfig{{Type: "mysql", TargetAddr: ":2"}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)
			if err := Validate(cfg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
