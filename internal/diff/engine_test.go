package diff

import (
	"strings"
	"testing"

	"shadiff/internal/model"
	"shadiff/internal/storage"
)

func newDiffStore(t *testing.T) (*storage.FileStore, *model.Session) {
	t.Helper()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	session := &model.Session{Name: "diff", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	return store, session
}

func TestEngineRun_UsesDefaultRulesAndIgnoreOrder(t *testing.T) {
	store, session := newDiffStore(t)

	original := &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"createdAt":"2024-01-01T10:00:00Z","items":[1,2]}`)},
	}
	replayed := &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   original.Request,
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"createdAt":"2025-05-05T12:00:00Z","items":[2,1]}`)},
	}
	if err := store.AppendRecord(session.ID, original); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, replayed); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	engine := NewEngine(store, EngineConfig{
		SessionID:   session.ID,
		IgnoreOrder: true,
	})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if !results[0].Match {
		t.Fatalf("expected diff result to match, got %+v", results[0].Differences)
	}
}

func TestEngineRun_MissingReplayRecordIsReported(t *testing.T) {
	store, session := newDiffStore(t)
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "replay-2",
		SessionID: session.ID,
		Sequence:  2,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	engine := NewEngine(store, EngineConfig{SessionID: session.ID})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].Differences) != 1 || !strings.Contains(results[0].Differences[0].Message, "replay record missing") {
		t.Fatalf("unexpected differences: %+v", results[0].Differences)
	}
}

func TestEngineRun_ReplayFailureIsReported(t *testing.T) {
	store, session := newDiffStore(t)
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Error:     "dial tcp 127.0.0.1:1: connect: connection refused",
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	engine := NewEngine(store, EngineConfig{SessionID: session.ID})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].Differences) != 1 || !strings.Contains(results[0].Differences[0].Message, "replay failed") {
		t.Fatalf("unexpected differences: %+v", results[0].Differences)
	}
}

func TestEngineRun_UsesSQLSideEffectComparer(t *testing.T) {
	store, session := newDiffStore(t)
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:          "orig-1",
		SessionID:   session.ID,
		Sequence:    1,
		Request:     model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:    model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
		SideEffects: []model.SideEffect{model.NewSQLSideEffect("mysql", "SELECT 1", 0)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:          "replay-1",
		SessionID:   session.ID,
		Sequence:    1,
		Request:     model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:    model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
		SideEffects: []model.SideEffect{model.NewSQLSideEffect("mysql", "SELECT 2", 0)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	engine := NewEngine(store, EngineConfig{SessionID: session.ID})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Match {
		t.Fatal("expected SQL side effect diff to break match")
	}
	if len(results[0].Differences) == 0 || results[0].Differences[0].Kind != model.DiffDBQuery {
		t.Fatalf("unexpected differences: %+v", results[0].Differences)
	}
}

func TestEngineRun_UsesMongoSideEffectComparer(t *testing.T) {
	store, session := newDiffStore(t)
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
		SideEffects: []model.SideEffect{model.NewMongoSideEffect(model.MongoSideEffect{
			Database:   "app",
			Collection: "users",
			Operation:  "find",
		}, 0)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:  model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
		SideEffects: []model.SideEffect{model.NewMongoSideEffect(model.MongoSideEffect{
			Database:   "app",
			Collection: "users",
			Operation:  "update",
		}, 0)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	engine := NewEngine(store, EngineConfig{SessionID: session.ID})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Match {
		t.Fatal("expected Mongo side effect diff to break match")
	}
	if len(results[0].Differences) == 0 || results[0].Differences[0].Kind != model.DiffMongoOp {
		t.Fatalf("unexpected differences: %+v", results[0].Differences)
	}
}

func TestEngineRun_UsesRedisSideEffectComparer(t *testing.T) {
	store, session := newDiffStore(t)
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:          "orig-1",
		SessionID:   session.ID,
		Sequence:    1,
		Request:     model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:    model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
		SideEffects: []model.SideEffect{model.NewRedisSideEffect("SET", "user:1", []string{"user:1", "old"}, 0)},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:          "replay-1",
		SessionID:   session.ID,
		Sequence:    1,
		Request:     model.HTTPRequest{Method: "GET", Path: "/items"},
		Response:    model.HTTPResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)},
		SideEffects: []model.SideEffect{model.NewRedisSideEffect("SET", "user:1", []string{"user:1", "new"}, 0)},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}

	engine := NewEngine(store, EngineConfig{SessionID: session.ID})
	results, err := engine.Run()
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Match {
		t.Fatal("expected Redis side effect diff to break match")
	}
	if len(results[0].Differences) == 0 || results[0].Differences[0].Kind != model.DiffRedisCommand {
		t.Fatalf("unexpected differences: %+v", results[0].Differences)
	}
}
