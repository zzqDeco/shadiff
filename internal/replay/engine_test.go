package replay

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"shadiff/internal/model"
	"shadiff/internal/storage"
)

func newReplayStore(t *testing.T) (*storage.FileStore, *model.Session) {
	t.Helper()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	session := &model.Session{Name: "replay", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	return store, session
}

func TestEngineRun_SavesReplayRecords(t *testing.T) {
	store, session := newReplayStore(t)
	original := &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request: model.HTTPRequest{
			Method: "GET",
			Path:   "/hello",
		},
	}
	if err := store.AppendRecord(session.ID, original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	engine := NewEngine(store, EngineConfig{
		SessionID: session.ID,
		TargetURL: server.URL,
	})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}

	replays, err := store.ListReplayRecords(session.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords() error: %v", err)
	}
	if len(replays) != 1 {
		t.Fatalf("len(replays) = %d, want 1", len(replays))
	}
	if replays[0].Response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", replays[0].Response.StatusCode, http.StatusCreated)
	}
}

func TestEngineRun_EmptySessionReturnsError(t *testing.T) {
	store, session := newReplayStore(t)
	engine := NewEngine(store, EngineConfig{
		SessionID: session.ID,
		TargetURL: "http://example.com",
	})

	if _, err := engine.Run(); err == nil {
		t.Fatal("expected empty session to return an error")
	}
}
