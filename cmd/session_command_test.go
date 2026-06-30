package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"shadiff/internal/model"
	"shadiff/internal/sessioninspect"

	"github.com/spf13/cobra"
)

func TestResolveSession_UsesLatestByName(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)

	first := &model.Session{Name: "demo", Status: model.SessionCompleted}
	second := &model.Session{Name: "demo", Status: model.SessionCompleted}
	if err := store.Create(first); err != nil {
		t.Fatalf("Create(first) error: %v", err)
	}
	if err := store.Create(second); err != nil {
		t.Fatalf("Create(second) error: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := store.Update(second); err != nil {
		t.Fatalf("Update(second) error: %v", err)
	}

	id, err := resolveSession(store, "demo")
	if err != nil {
		t.Fatalf("resolveSession() error: %v", err)
	}
	if id != second.ID {
		t.Fatalf("resolved id = %q, want %q", id, second.ID)
	}
}

func TestFindSession_AmbiguousNameReturnsError(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)

	for i := 0; i < 2; i++ {
		if err := store.Create(&model.Session{Name: "same", Status: model.SessionCompleted}); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	if _, err := findSession(store, "same"); err == nil {
		t.Fatal("expected ambiguous session name to return an error")
	}
}

func TestRunSessionListShowDelete(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "session-case", Status: model.SessionCompleted}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	listOutput := captureStdout(t, func() {
		if err := runSessionList(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runSessionList() error: %v", err)
		}
	})
	if !strings.Contains(listOutput, session.Name) {
		t.Fatalf("list output = %q, want session name", listOutput)
	}

	showOutput := captureStdout(t, func() {
		if err := runSessionShow(&cobra.Command{}, []string{session.ID}); err != nil {
			t.Fatalf("runSessionShow() error: %v", err)
		}
	})
	if !strings.Contains(showOutput, "ID:") {
		t.Fatalf("show output = %q, want session details", showOutput)
	}

	deleteOutput := captureStdout(t, func() {
		if err := runSessionDelete(&cobra.Command{}, []string{session.ID}); err != nil {
			t.Fatalf("runSessionDelete() error: %v", err)
		}
	})
	if !strings.Contains(deleteOutput, "Session deleted") {
		t.Fatalf("delete output = %q, want delete message", deleteOutput)
	}
}

func TestRunSessionInspect_PrintsArtifactAndSideEffectCounts(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "inspect-case", Status: model.SessionReplayed}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
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
		t.Fatalf("AppendRecord() error: %v", err)
	}
	if err := store.AppendReplayRecord(session.ID, &model.Record{
		ID:        "replay-1",
		SessionID: session.ID,
		Sequence:  1,
		SideEffects: []model.SideEffect{
			model.NewRedisSideEffect("GET", "user:1", []string{"user:1"}, 0),
		},
	}); err != nil {
		t.Fatalf("AppendReplayRecord() error: %v", err)
	}
	if err := store.SaveResults(session.ID, []model.DiffResult{{RecordID: "record-1", Sequence: 1}}); err != nil {
		t.Fatalf("SaveResults() error: %v", err)
	}

	sessionInspectFormat = "terminal"
	output := captureStdout(t, func() {
		if err := runSessionInspect(&cobra.Command{}, []string{session.ID}); err != nil {
			t.Fatalf("runSessionInspect() error: %v", err)
		}
	})
	for _, want := range []string{
		"Records: 1",
		"Replay records: 1",
		"Diff results: 1",
		"mysql: 1",
		"mongo: 1",
		"redis: 1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("inspect output missing %q:\n%s", want, output)
		}
	}
}

func TestRunSessionInspect_JSONWarningsForMissingReplayAndDiff(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "inspect-missing", Status: model.SessionCompleted}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{ID: "record-1", SessionID: session.ID}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	sessionInspectFormat = "json"
	var out strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runSessionInspect(cmd, []string{session.Name}); err != nil {
		t.Fatalf("runSessionInspect() error: %v", err)
	}

	var report sessioninspect.Report
	if err := json.Unmarshal([]byte(out.String()), &report); err != nil {
		t.Fatalf("Unmarshal() error = %v\noutput=%s", err, out.String())
	}
	if report.RecordCount != 1 || report.ReplayRecordCount != 0 || report.DiffResultCount != 0 {
		t.Fatalf("unexpected counts: %+v", report)
	}
	if len(report.Warnings) != 2 {
		t.Fatalf("warnings = %v, want replay and diff warnings", report.Warnings)
	}
}
