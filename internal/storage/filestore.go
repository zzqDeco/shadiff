package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"shadiff/internal/model"

	"github.com/google/uuid"
)

// FileStore is a file-system-based storage implementation
// Directory structure: {baseDir}/sessions/{id}/session.json, records.jsonl, replay-records.jsonl, diff-results.json
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStore creates a new file store instance
func NewFileStore(baseDir string) (*FileStore, error) {
	dir := filepath.Join(baseDir, "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}
	return &FileStore{baseDir: dir}, nil
}

// --- SessionStore implementation ---

// Create creates a new session
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

// Get retrieves a session by ID
func (fs *FileStore) Get(id string) (*model.Session, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.loadSession(id)
}

// List lists all sessions with optional filtering
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

		// Apply filter conditions
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

	// Sort by update time descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})
	return sessions, nil
}

// Update updates session metadata
func (fs *FileStore) Update(session *model.Session) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	session.UpdatedAt = time.Now().UnixMilli()
	return fs.saveSession(session)
}

// Delete deletes a session and all its data
func (fs *FileStore) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	dir := filepath.Join(fs.baseDir, id)
	return os.RemoveAll(dir)
}

// --- RecordStore implementation ---

// AppendRecord appends a record to the JSONL file
func (fs *FileStore) AppendRecord(sessionID string, record *model.Record) error {
	return fs.appendRecord(sessionID, "records.jsonl", record)
}

// AppendReplayRecord appends a replay record to the JSONL file
func (fs *FileStore) AppendReplayRecord(sessionID string, record *model.Record) error {
	return fs.appendRecord(sessionID, "replay-records.jsonl", record)
}

func (fs *FileStore) appendRecord(sessionID, filename string, record *model.Record) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.baseDir, sessionID, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", filename, err)
	}
	defer f.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// ListRecords reads all recorded records for a session
func (fs *FileStore) ListRecords(sessionID string) ([]model.Record, error) {
	return fs.listRecords(sessionID, "records.jsonl")
}

// ListReplayRecords reads all replay records for a session
func (fs *FileStore) ListReplayRecords(sessionID string) ([]model.Record, error) {
	return fs.listRecords(sessionID, "replay-records.jsonl")
}

func (fs *FileStore) listRecords(sessionID, filename string) ([]model.Record, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := filepath.Join(fs.baseDir, sessionID, filename)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var records []model.Record
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // max 10MB per line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec model.Record
		if err := json.Unmarshal(line, &rec); err != nil {
			continue // skip corrupted lines
		}
		records = append(records, rec)
	}
	return records, scanner.Err()
}

// GetRecord retrieves a single record
func (fs *FileStore) GetRecord(sessionID string, recordID string) (*model.Record, error) {
	records, err := fs.ListRecords(sessionID)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == recordID {
			return &records[i], nil
		}
	}
	return nil, fmt.Errorf("record %s not found", recordID)
}

// CountRecords returns the number of records in a session
func (fs *FileStore) CountRecords(sessionID string) (int, error) {
	records, err := fs.ListRecords(sessionID)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// --- DiffStore implementation ---

// SaveResults saves diff results
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

// LoadResults loads diff results
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

// --- Internal methods ---

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
