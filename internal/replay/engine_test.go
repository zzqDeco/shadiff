package replay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"shadiff/internal/model"
	"shadiff/internal/storage"
)

type fakeEngineFlusher struct {
	flush func(context.Context) error
}

func (f fakeEngineFlusher) Flush(ctx context.Context) error {
	if f.flush == nil {
		return nil
	}
	return f.flush(ctx)
}

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

func TestEngineRun_PersistsFailedReplayRecords(t *testing.T) {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	targetURL := server.URL
	server.Close()

	engine := NewEngine(store, EngineConfig{
		SessionID: session.ID,
		TargetURL: targetURL,
		Timeout:   100 * time.Millisecond,
	})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 || results[0].Error == nil {
		t.Fatalf("unexpected replay results: %+v", results)
	}

	replays, err := store.ListReplayRecords(session.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords() error: %v", err)
	}
	if len(replays) != 1 {
		t.Fatalf("len(replays) = %d, want 1", len(replays))
	}
	if replays[0].Error == "" {
		t.Fatal("expected persisted replay record to include error")
	}
}

func TestEngineRun_AttachesReplayDBSideEffects(t *testing.T) {
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

	sideEffects := make(chan model.SideEffect, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sideEffects <- model.NewSQLSideEffect("mysql", "SELECT 1", time.Now().UnixMilli())
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	engine := NewEngine(store, EngineConfig{
		SessionID:    session.ID,
		TargetURL:    server.URL,
		SideEffectCh: sideEffects,
	})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].Replayed.SideEffects) != 1 {
		t.Fatalf("replayed sideEffects len = %d, want 1", len(results[0].Replayed.SideEffects))
	}

	replays, err := store.ListReplayRecords(session.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords() error: %v", err)
	}
	if len(replays) != 1 || len(replays[0].SideEffects) != 1 {
		t.Fatalf("persisted replay sideEffects = %+v", replays)
	}
}

func TestEngineRun_FlushesReplaySideEffectsBeforeWindowClose(t *testing.T) {
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

	var (
		mu         sync.Mutex
		lateEffect model.SideEffect
	)
	sideEffects := make(chan model.SideEffect, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		lateEffect = model.NewSQLSideEffect("mysql", "SELECT late", time.Now().UnixMilli())
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	engine := NewEngine(store, EngineConfig{
		SessionID:    session.ID,
		TargetURL:    server.URL,
		SideEffectCh: sideEffects,
		Flusher: fakeEngineFlusher{flush: func(ctx context.Context) error {
			mu.Lock()
			effect := lateEffect
			mu.Unlock()
			sideEffects <- effect
			return nil
		}},
		FlushTimeout: 50 * time.Millisecond,
	})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].Replayed.SideEffects) != 1 {
		t.Fatalf("replayed sideEffects len = %d, want 1", len(results[0].Replayed.SideEffects))
	}
	if results[0].Replayed.SideEffects[0].SQL().Query != "SELECT late" {
		t.Fatalf("side effect query = %q, want %q", results[0].Replayed.SideEffects[0].SQL().Query, "SELECT late")
	}
}

func TestEngineRun_FlushTimeoutDoesNotFailReplay(t *testing.T) {
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
		SessionID:    session.ID,
		TargetURL:    server.URL,
		SideEffectCh: make(chan model.SideEffect, 1),
		Flusher: fakeEngineFlusher{flush: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}},
		FlushTimeout: time.Millisecond,
	})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 || results[0].Error != nil {
		t.Fatalf("unexpected replay results: %+v", results)
	}
}
