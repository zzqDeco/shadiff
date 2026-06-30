package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SaveRequestBodyArtifact persists the full captured request body for a record and returns
// a session-relative artifact path suitable for model.HTTPRequest.BodyRef.
func (fs *FileStore) SaveRequestBodyArtifact(sessionID, recordID string, src io.Reader) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	relPath := filepath.Join("artifacts", "request-bodies", recordID+".bin")
	fullPath := filepath.Join(fs.baseDir, sessionID, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("create request body artifact dir: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create request body artifact: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, src); err != nil {
		return "", fmt.Errorf("write request body artifact: %w", err)
	}

	return filepath.ToSlash(relPath), nil
}

// OpenRequestBodyArtifact opens a previously stored request body artifact using its session-relative path.
func (fs *FileStore) OpenRequestBodyArtifact(sessionID, ref string) (io.ReadCloser, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	fullPath, err := fs.requestBodyArtifactPath(sessionID, ref)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open request body artifact: %w", err)
	}
	return f, nil
}
