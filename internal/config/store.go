package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store is a thread-safe configuration store, persisted to ~/.shadiff/config.json
type Store struct {
	path   string
	config *AppConfig
	mu     sync.RWMutex
}

// NewStore creates a config store instance, automatically loading or initializing default config.
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(homeDir, ".shadiff")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		path: filepath.Join(dir, "config.json"),
	}
	if err := s.Load(); err != nil {
		s.config = DefaultConfig()
	}
	return s, nil
}

// Load reads config from file.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}
	s.config = cfg
	return nil
}

// Save writes config to file.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Get returns a copy of the config.
func (s *Store) Get() *AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := *s.config
	return &cfg
}

// Update atomically updates the config and persists it.
func (s *Store) Update(fn func(*AppConfig)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.config)
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// DataDir returns the data directory path, preferring the configured DataDir, otherwise defaults to ~/.shadiff
func (s *Store) DataDir() string {
	cfg := s.Get()
	if cfg.Storage.DataDir != "" {
		return cfg.Storage.DataDir
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".shadiff")
}
