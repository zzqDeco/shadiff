package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"shadiff/internal/model"
)

// AppendRecord appends a record to the JSONL file.
func (fs *FileStore) AppendRecord(sessionID string, record *model.Record) error {
	return fs.appendRecord(sessionID, "records.jsonl", record)
}

// AppendReplayRecord appends a replay record to the JSONL file.
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

// ListRecords reads all recorded records for a session.
func (fs *FileStore) ListRecords(sessionID string) ([]model.Record, error) {
	return fs.listRecords(sessionID, "records.jsonl")
}

// ListReplayRecords reads all replay records for a session.
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

// GetRecord retrieves a single record.
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

// CountRecords returns the number of records in a session.
func (fs *FileStore) CountRecords(sessionID string) (int, error) {
	records, err := fs.ListRecords(sessionID)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}
