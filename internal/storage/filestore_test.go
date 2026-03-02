package storage

import (
	"os"
	"path/filepath"
	"testing"

	"shadiff/internal/model"
)

// newTestStore creates a FileStore backed by a temporary directory.
// The caller does not need to clean up; t.TempDir() handles removal.
func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	return fs
}

func makeSession(name string, status model.SessionStatus) *model.Session {
	return &model.Session{
		Name:   name,
		Status: status,
		Source: model.EndpointConfig{BaseURL: "http://localhost:8080"},
		Target: model.EndpointConfig{BaseURL: "http://localhost:9090"},
	}
}

func makeRecord(id string, seq int) *model.Record {
	return &model.Record{
		ID:       id,
		Sequence: seq,
		Request: model.HTTPRequest{
			Method: "GET",
			Path:   "/api/test",
		},
		Response: model.HTTPResponse{
			StatusCode: 200,
			Body:       []byte(`{"ok":true}`),
		},
	}
}

// ===================== Session Tests =====================

func TestCreateAndGetSession(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("test-session", model.SessionRecording)

	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sess.ID == "" {
		t.Fatal("expected session ID to be assigned")
	}
	if sess.CreatedAt == 0 {
		t.Fatal("expected CreatedAt to be set")
	}
	if sess.UpdatedAt == 0 {
		t.Fatal("expected UpdatedAt to be set")
	}

	got, err := fs.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test-session" {
		t.Errorf("Name = %q, want %q", got.Name, "test-session")
	}
	if got.Status != model.SessionRecording {
		t.Errorf("Status = %q, want %q", got.Status, model.SessionRecording)
	}
}

func TestCreateSession_AssignsDefaults(t *testing.T) {
	fs := newTestStore(t)
	sess := &model.Session{Name: "defaults"}

	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sess.Tags == nil {
		t.Error("expected Tags to be initialized, got nil")
	}
	if sess.Metadata == nil {
		t.Error("expected Metadata to be initialized, got nil")
	}
}

func TestCreateSession_PreservesExistingID(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("custom-id", model.SessionRecording)
	sess.ID = "myid1234"

	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.ID != "myid1234" {
		t.Errorf("ID = %q, want %q", sess.ID, "myid1234")
	}

	got, err := fs.Get("myid1234")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "custom-id" {
		t.Errorf("Name = %q, want %q", got.Name, "custom-id")
	}
}

func TestGetSession_NonExistent(t *testing.T) {
	fs := newTestStore(t)
	_, err := fs.Get("does-not-exist")
	if err == nil {
		t.Fatal("expected error for non-existent session, got nil")
	}
}

func TestUpdateSession(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("to-update", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sess.Status = model.SessionCompleted
	sess.Description = "updated description"
	if err := fs.Update(sess); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := fs.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != model.SessionCompleted {
		t.Errorf("Status = %q, want %q", got.Status, model.SessionCompleted)
	}
	if got.Description != "updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "updated description")
	}
	if got.UpdatedAt == 0 {
		t.Error("expected UpdatedAt to be set after Update")
	}
}

func TestDeleteSession(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("to-delete", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := fs.Delete(sess.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := fs.Get(sess.ID)
	if err == nil {
		t.Fatal("expected error after deleting session, got nil")
	}

	// Verify the directory was removed
	dir := filepath.Join(fs.baseDir, sess.ID)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected session directory to be removed, stat err = %v", err)
	}
}

func TestDeleteSession_NonExistent(t *testing.T) {
	fs := newTestStore(t)
	// os.RemoveAll on a non-existent path returns nil
	if err := fs.Delete("no-such-id"); err != nil {
		t.Fatalf("Delete non-existent: unexpected error %v", err)
	}
}

func TestListSessions_Empty(t *testing.T) {
	fs := newTestStore(t)
	sessions, err := fs.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestListSessions_NoFilter(t *testing.T) {
	fs := newTestStore(t)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		s := makeSession(name, model.SessionRecording)
		if err := fs.Create(s); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	sessions, err := fs.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestListSessions_FilterByStatus(t *testing.T) {
	fs := newTestStore(t)
	s1 := makeSession("rec1", model.SessionRecording)
	s2 := makeSession("done1", model.SessionCompleted)
	s3 := makeSession("done2", model.SessionCompleted)
	for _, s := range []*model.Session{s1, s2, s3} {
		if err := fs.Create(s); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	sessions, err := fs.List(&model.SessionFilter{Status: "completed"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 completed sessions, got %d", len(sessions))
	}
	for _, s := range sessions {
		if s.Status != model.SessionCompleted {
			t.Errorf("expected status completed, got %q", s.Status)
		}
	}
}

func TestListSessions_FilterByName(t *testing.T) {
	fs := newTestStore(t)
	s1 := makeSession("my-api-test", model.SessionRecording)
	s2 := makeSession("api-regression", model.SessionRecording)
	s3 := makeSession("unrelated", model.SessionRecording)
	for _, s := range []*model.Session{s1, s2, s3} {
		if err := fs.Create(s); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	sessions, err := fs.List(&model.SessionFilter{Name: "api"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions matching 'api', got %d", len(sessions))
	}
}

func TestListSessions_FilterByTags(t *testing.T) {
	fs := newTestStore(t)
	s1 := makeSession("tagged1", model.SessionRecording)
	s1.Tags = []string{"v1", "smoke"}
	s2 := makeSession("tagged2", model.SessionRecording)
	s2.Tags = []string{"v2"}
	s3 := makeSession("no-tags", model.SessionRecording)

	for _, s := range []*model.Session{s1, s2, s3} {
		if err := fs.Create(s); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	sessions, err := fs.List(&model.SessionFilter{Tags: []string{"v1"}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session with tag 'v1', got %d", len(sessions))
	}
	if len(sessions) > 0 && sessions[0].Name != "tagged1" {
		t.Errorf("expected session 'tagged1', got %q", sessions[0].Name)
	}
}

// ===================== Record Tests =====================

func TestAppendAndListRecords(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("rec-session", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	r1 := makeRecord("r1", 1)
	r2 := makeRecord("r2", 2)
	r1.SessionID = sess.ID
	r2.SessionID = sess.ID

	if err := fs.AppendRecord(sess.ID, r1); err != nil {
		t.Fatalf("AppendRecord r1: %v", err)
	}
	if err := fs.AppendRecord(sess.ID, r2); err != nil {
		t.Fatalf("AppendRecord r2: %v", err)
	}

	records, err := fs.ListRecords(sess.ID)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "r1" {
		t.Errorf("records[0].ID = %q, want %q", records[0].ID, "r1")
	}
	if records[1].ID != "r2" {
		t.Errorf("records[1].ID = %q, want %q", records[1].ID, "r2")
	}
}

func TestListRecords_NoRecordsFile(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("empty-rec", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	records, err := fs.ListRecords(sess.ID)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil for missing records file, got %v", records)
	}
}

func TestCountRecords(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("count-session", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Count with no records
	count, err := fs.CountRecords(sess.ID)
	if err != nil {
		t.Fatalf("CountRecords: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 records, got %d", count)
	}

	// Append records and count again
	for i := 0; i < 5; i++ {
		r := makeRecord("r", i)
		if err := fs.AppendRecord(sess.ID, r); err != nil {
			t.Fatalf("AppendRecord: %v", err)
		}
	}

	count, err = fs.CountRecords(sess.ID)
	if err != nil {
		t.Fatalf("CountRecords: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 records, got %d", count)
	}
}

func TestGetRecord(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("get-rec", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	r1 := makeRecord("find-me", 1)
	r2 := makeRecord("other", 2)
	if err := fs.AppendRecord(sess.ID, r1); err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}
	if err := fs.AppendRecord(sess.ID, r2); err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}

	got, err := fs.GetRecord(sess.ID, "find-me")
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	if got.ID != "find-me" {
		t.Errorf("ID = %q, want %q", got.ID, "find-me")
	}
	if got.Sequence != 1 {
		t.Errorf("Sequence = %d, want 1", got.Sequence)
	}
}

func TestGetRecord_NotFound(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("missing-rec", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	r1 := makeRecord("exists", 1)
	if err := fs.AppendRecord(sess.ID, r1); err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}

	_, err := fs.GetRecord(sess.ID, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent record, got nil")
	}
}

// ===================== Replay Record Tests =====================

func TestAppendAndListReplayRecords(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("replay-session", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	r1 := makeRecord("rp1", 1)
	r2 := makeRecord("rp2", 2)

	if err := fs.AppendReplayRecord(sess.ID, r1); err != nil {
		t.Fatalf("AppendReplayRecord r1: %v", err)
	}
	if err := fs.AppendReplayRecord(sess.ID, r2); err != nil {
		t.Fatalf("AppendReplayRecord r2: %v", err)
	}

	records, err := fs.ListReplayRecords(sess.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 replay records, got %d", len(records))
	}
	if records[0].ID != "rp1" {
		t.Errorf("records[0].ID = %q, want %q", records[0].ID, "rp1")
	}
}

func TestReplayRecords_Independent(t *testing.T) {
	// Verify that records and replay records are stored separately
	fs := newTestStore(t)
	sess := makeSession("independent", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rec := makeRecord("normal", 1)
	rep := makeRecord("replay", 1)

	if err := fs.AppendRecord(sess.ID, rec); err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}
	if err := fs.AppendReplayRecord(sess.ID, rep); err != nil {
		t.Fatalf("AppendReplayRecord: %v", err)
	}

	records, err := fs.ListRecords(sess.ID)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(records) != 1 || records[0].ID != "normal" {
		t.Errorf("ListRecords should return only 'normal' record, got %v", records)
	}

	replays, err := fs.ListReplayRecords(sess.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords: %v", err)
	}
	if len(replays) != 1 || replays[0].ID != "replay" {
		t.Errorf("ListReplayRecords should return only 'replay' record, got %v", replays)
	}
}

func TestListReplayRecords_NoFile(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("no-replay", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	records, err := fs.ListReplayRecords(sess.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil for missing replay-records file, got %v", records)
	}
}

// ===================== DiffResult Tests =====================

func TestSaveAndLoadResults(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("diff-session", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	results := []model.DiffResult{
		{
			RecordID: "r1",
			Sequence: 1,
			Match:    true,
			Request: model.HTTPRequest{
				Method: "GET",
				Path:   "/api/v1/users",
			},
		},
		{
			RecordID: "r2",
			Sequence: 2,
			Match:    false,
			Request: model.HTTPRequest{
				Method: "POST",
				Path:   "/api/v1/users",
			},
			Differences: []model.Difference{
				{
					Kind:     model.DiffStatusCode,
					Path:     "statusCode",
					Expected: 200,
					Actual:   500,
					Message:  "status code mismatch",
					Severity: model.SeverityError,
				},
			},
		},
	}

	if err := fs.SaveResults(sess.ID, results); err != nil {
		t.Fatalf("SaveResults: %v", err)
	}

	loaded, err := fs.LoadResults(sess.ID)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 diff results, got %d", len(loaded))
	}
	if loaded[0].RecordID != "r1" || loaded[0].Match != true {
		t.Errorf("loaded[0] mismatch: %+v", loaded[0])
	}
	if loaded[1].RecordID != "r2" || loaded[1].Match != false {
		t.Errorf("loaded[1] mismatch: %+v", loaded[1])
	}
	if len(loaded[1].Differences) != 1 {
		t.Fatalf("expected 1 difference, got %d", len(loaded[1].Differences))
	}
	diff := loaded[1].Differences[0]
	if diff.Kind != model.DiffStatusCode {
		t.Errorf("diff.Kind = %q, want %q", diff.Kind, model.DiffStatusCode)
	}
	if diff.Severity != model.SeverityError {
		t.Errorf("diff.Severity = %q, want %q", diff.Severity, model.SeverityError)
	}
}

func TestLoadResults_NoFile(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("no-diff", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	results, err := fs.LoadResults(sess.ID)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for missing diff-results file, got %v", results)
	}
}

func TestSaveResults_Overwrite(t *testing.T) {
	fs := newTestStore(t)
	sess := makeSession("overwrite-diff", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	first := []model.DiffResult{{RecordID: "old", Sequence: 1, Match: true}}
	if err := fs.SaveResults(sess.ID, first); err != nil {
		t.Fatalf("SaveResults (first): %v", err)
	}

	second := []model.DiffResult{
		{RecordID: "new1", Sequence: 1, Match: false},
		{RecordID: "new2", Sequence: 2, Match: true},
	}
	if err := fs.SaveResults(sess.ID, second); err != nil {
		t.Fatalf("SaveResults (second): %v", err)
	}

	loaded, err := fs.LoadResults(sess.ID)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 results after overwrite, got %d", len(loaded))
	}
	if loaded[0].RecordID != "new1" {
		t.Errorf("expected first result to be 'new1', got %q", loaded[0].RecordID)
	}
}

// ===================== Integration / Lifecycle Test =====================

func TestSessionLifecycle(t *testing.T) {
	fs := newTestStore(t)

	// Create a session
	sess := makeSession("lifecycle", model.SessionRecording)
	if err := fs.Create(sess); err != nil {
		t.Fatalf("Create: %v", err)
	}
	id := sess.ID

	// Append records
	for i := 1; i <= 3; i++ {
		r := makeRecord("rec", i)
		r.SessionID = id
		if err := fs.AppendRecord(id, r); err != nil {
			t.Fatalf("AppendRecord seq=%d: %v", i, err)
		}
	}

	count, err := fs.CountRecords(id)
	if err != nil {
		t.Fatalf("CountRecords: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}

	// Update session status
	sess.Status = model.SessionCompleted
	if err := fs.Update(sess); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := fs.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != model.SessionCompleted {
		t.Errorf("Status = %q, want %q", got.Status, model.SessionCompleted)
	}

	// Delete the session
	if err := fs.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = fs.Get(id)
	if err == nil {
		t.Fatal("expected error after Delete, got nil")
	}
}
