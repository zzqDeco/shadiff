package sessioninspect

import (
	"bytes"
	"strings"
	"testing"

	"shadiff/internal/model"
	"shadiff/internal/storage"
)

func TestBuildReport_CountsArtifactsAndSideEffects(t *testing.T) {
	dataDir := t.TempDir()
	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session := &model.Session{Name: "inspect-case", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "record-1",
		SessionID: session.ID,
		Sequence:  1,
		SideEffects: []model.SideEffect{
			model.NewSQLSideEffect("mysql", "SELECT 1", 0),
			model.NewMongoSideEffect(model.MongoSideEffect{Database: "app", Collection: "users", Operation: "find"}, 0),
		},
	}); err != nil {
		t.Fatalf("AppendRecord() error = %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		SideEffects: []model.SideEffect{
			model.NewRedisSideEffect("GET", "user:1", []string{"user:1"}, 0),
		},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error = %v", err)
	}
	if err := store.SaveResults(session.ID, []model.DiffResult{{RecordID: "record-1", Sequence: 1}}); err != nil {
		t.Fatalf("SaveResults() error = %v", err)
	}

	report, err := BuildReport(store, dataDir, session.ID)
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if report.RecordCount != 1 || report.ReplayRecordCount != 1 || report.DiffResultCount != 1 {
		t.Fatalf("unexpected counts: %+v", report)
	}
	if report.RecordSideEffects["mysql"] != 1 || report.RecordSideEffects["mongo"] != 1 {
		t.Fatalf("record side effect counts = %+v", report.RecordSideEffects)
	}
	if report.ReplaySideEffects["redis"] != 1 {
		t.Fatalf("replay side effect counts = %+v", report.ReplaySideEffects)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("warnings = %v, want none", report.Warnings)
	}
}

func TestBuildReport_WarnsWhenReplayAndDiffMissing(t *testing.T) {
	dataDir := t.TempDir()
	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session := &model.Session{Name: "inspect-missing", Status: model.SessionCompleted}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{ID: "record-1", SessionID: session.ID}); err != nil {
		t.Fatalf("AppendRecord() error = %v", err)
	}

	report, err := BuildReport(store, dataDir, session.ID)
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(report.Warnings) != 2 {
		t.Fatalf("warnings = %v, want replay and diff warnings", report.Warnings)
	}
}

func TestPrintReportIncludesCounts(t *testing.T) {
	report := Report{
		Session:           model.Session{ID: "abc123", Name: "demo", Status: model.SessionReplayed},
		DataDir:           "/tmp/shadiff",
		SessionDir:        "/tmp/shadiff/sessions/abc123",
		Files:             map[string]FileStatus{"session": {Path: "/tmp/shadiff/sessions/abc123/session.json", Exists: true, Size: 10}},
		RecordCount:       1,
		ReplayRecordCount: 1,
		DiffResultCount:   1,
		RecordSideEffects: map[string]int{"mysql": 1, "redis": 0},
		ReplaySideEffects: map[string]int{"mysql": 0, "redis": 1},
	}

	var out bytes.Buffer
	PrintReport(&out, report)
	for _, want := range []string{"Session: abc123 (demo)", "Records: 1", "mysql: 1", "redis: 1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}
