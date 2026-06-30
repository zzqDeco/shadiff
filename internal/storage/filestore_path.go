package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (fs *FileStore) requestBodyArtifactPath(sessionID, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("request body artifact path is empty")
	}

	clean := filepath.Clean(filepath.FromSlash(ref))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid request body artifact path: %s", ref)
	}

	fullPath := filepath.Join(fs.baseDir, sessionID, clean)
	sessionRoot := filepath.Join(fs.baseDir, sessionID)
	rel, err := filepath.Rel(sessionRoot, fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve request body artifact path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid request body artifact path: %s", ref)
	}

	return fullPath, nil
}
