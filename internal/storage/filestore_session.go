package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"shadiff/internal/model"

	"github.com/google/uuid"
)

// Create creates a new session.
func (fs *FileStore) Create(session *model.Session) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if session.ID == "" {
		session.ID = uuid.New().String()[:8]
	}
	now := time.Now().UnixMilli()
	if session.CreatedAt == 0 {
		session.CreatedAt = now
	}
	session.UpdatedAt = now

	if session.Tags == nil {
		session.Tags = []string{}
	}
	if session.Metadata == nil {
		session.Metadata = map[string]string{}
	}

	dir := filepath.Join(fs.baseDir, session.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	return fs.saveSession(session)
}

// Get retrieves a session by ID.
func (fs *FileStore) Get(id string) (*model.Session, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.loadSession(id)
}

// List lists all sessions with optional filtering.
func (fs *FileStore) List(filter *model.SessionFilter) ([]model.Session, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return nil, err
	}

	var sessions []model.Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sess, err := fs.loadSession(entry.Name())
		if err != nil {
			continue // skip corrupted sessions
		}

		if filter != nil {
			if filter.Name != "" && !strings.Contains(sess.Name, filter.Name) {
				continue
			}
			if filter.Status != "" && string(sess.Status) != filter.Status {
				continue
			}
			if len(filter.Tags) > 0 && !hasAnyTag(sess.Tags, filter.Tags) {
				continue
			}
		}

		sessions = append(sessions, *sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})
	return sessions, nil
}

// Update updates session metadata.
func (fs *FileStore) Update(session *model.Session) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	session.UpdatedAt = time.Now().UnixMilli()
	return fs.saveSession(session)
}

// Delete deletes a session and all its data.
func (fs *FileStore) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	dir := filepath.Join(fs.baseDir, id)
	return os.RemoveAll(dir)
}

// PruneOldest removes the oldest non-recording sessions until the total number
// of sessions is at or below maxSessions.
func (fs *FileStore) PruneOldest(maxSessions int) error {
	if maxSessions < 1 {
		return nil
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return err
	}

	var sessions []model.Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sess, err := fs.loadSession(entry.Name())
		if err != nil {
			continue
		}
		sessions = append(sessions, *sess)
	}

	if len(sessions) <= maxSessions {
		return nil
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt < sessions[j].UpdatedAt
	})

	toDelete := len(sessions) - maxSessions
	for _, sess := range sessions {
		if toDelete == 0 {
			break
		}
		if sess.Status == model.SessionRecording {
			continue
		}
		if err := os.RemoveAll(filepath.Join(fs.baseDir, sess.ID)); err != nil {
			return err
		}
		toDelete--
	}

	return nil
}

func (fs *FileStore) loadSession(id string) (*model.Session, error) {
	path := filepath.Join(fs.baseDir, id, "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sess model.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	if sess.Tags == nil {
		sess.Tags = []string{}
	}
	if sess.Metadata == nil {
		sess.Metadata = map[string]string{}
	}
	return &sess, nil
}

func (fs *FileStore) saveSession(sess *model.Session) error {
	path := filepath.Join(fs.baseDir, sess.ID, "session.json")
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func hasAnyTag(sessionTags, filterTags []string) bool {
	tagSet := make(map[string]struct{}, len(sessionTags))
	for _, t := range sessionTags {
		tagSet[t] = struct{}{}
	}
	for _, t := range filterTags {
		if _, ok := tagSet[t]; ok {
			return true
		}
	}
	return false
}
