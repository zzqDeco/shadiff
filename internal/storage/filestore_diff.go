package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"shadiff/internal/model"
)

// SaveResults saves diff results.
func (fs *FileStore) SaveResults(sessionID string, results []model.DiffResult) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.baseDir, sessionID, "diff-results.json")
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal diff results: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadResults loads diff results.
func (fs *FileStore) LoadResults(sessionID string) ([]model.DiffResult, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.baseDir, sessionID, "diff-results.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []model.DiffResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("unmarshal diff results: %w", err)
	}
	return results, nil
}
