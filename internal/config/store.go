package config

import (
	"encoding/json"
	"errors"
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
	return NewStoreWithPath("")
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(homeDir, ".shadiff")
	return filepath.Join(dir, "config.json"), nil
}

// NewStoreWithPath creates a config store for the provided path.
// If path is empty, ~/.shadiff/config.json is used.
// Missing config files are initialized from defaults and written to disk.
func NewStoreWithPath(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		path:   path,
		config: DefaultConfig(),
	}
	if err := s.Load(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := s.Save(); err != nil {
			return nil, err
		}
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
