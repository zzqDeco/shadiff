package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStore is a file-system-based storage implementation.
// Directory structure: {baseDir}/sessions/{id}/session.json, records.jsonl, replay-records.jsonl, diff-results.json.
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStore creates a new file store instance.
func NewFileStore(baseDir string) (*FileStore, error) {
	dir := filepath.Join(baseDir, "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}
	return &FileStore{baseDir: dir}, nil
}
