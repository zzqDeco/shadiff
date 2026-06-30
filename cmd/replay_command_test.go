package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shadiff/internal/capture/dbhook"
	"shadiff/internal/config"
	"shadiff/internal/model"

	"github.com/spf13/cobra"
)

func TestRunReplay_ReplaysAndUpdatesSession(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "replay-case", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/hello"},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	replaySession = session.Name
	replayTarget = server.URL
	replayConcurrency = 0
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")

	if err := runReplay(cmd, nil); err != nil {
		t.Fatalf("runReplay() error: %v", err)
	}

	updated, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if updated.Status != model.SessionReplayed {
		t.Fatalf("session status = %q, want %q", updated.Status, model.SessionReplayed)
	}

	replays, err := store.ListReplayRecords(session.ID)
	if err != nil {
		t.Fatalf("ListReplayRecords() error: %v", err)
	}
	if len(replays) != 1 {
		t.Fatalf("len(replays) = %d, want 1", len(replays))
	}
}

func TestRunReplay_ConfigDBProxyRequiresSerialConcurrency(t *testing.T) {
	withRuntimeConfig(t, func(cfg *config.AppConfig) {
		cfg.Replay.Concurrency = 2
		cfg.Replay.DBProxies = []config.DBProxyConfig{
			{Type: "mysql", ListenAddr: ":13307", TargetAddr: "127.0.0.1:3306"},
		}
	})

	replaySession = ""
	replayTarget = "http://example.com"
	replayConcurrency = 0
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")

	err := runReplay(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "require concurrency 1") {
		t.Fatalf("runReplay() error = %v, want config concurrency validation", err)
	}
}

func TestRunReplay_DBProxyRequiresSerialConcurrency(t *testing.T) {
	withRuntimeConfig(t, nil)

	replaySession = ""
	replayTarget = "http://example.com"
	replayConcurrency = 2
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")
	if err := cmd.Flags().Set("concurrency", "2"); err != nil {
		t.Fatalf("Set(concurrency) error: %v", err)
	}
	if err := cmd.Flags().Set("db-proxy", "mysql://:13307->127.0.0.1:3306"); err != nil {
		t.Fatalf("Set(db-proxy) error: %v", err)
	}

	err := runReplay(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "require concurrency 1") {
		t.Fatalf("runReplay() error = %v, want concurrency validation", err)
	}
}

func TestRunReplay_DBProxyStartsHooks(t *testing.T) {
	withRuntimeConfig(t, nil)
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "replay-db-proxy", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/hello"},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	fakeHook := &fakeDBHook{sideEffects: make(chan model.SideEffect)}
	oldFactory := newDBHook
	defer func() { newDBHook = oldFactory }()
	newDBHook = func(cfg dbhook.Config) (dbhook.DBHook, error) {
		return fakeHook, nil
	}

	replaySession = session.Name
	replayTarget = server.URL
	replayConcurrency = 1
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")
	if err := cmd.Flags().Set("db-proxy", "mysql://:13307->127.0.0.1:3306"); err != nil {
		t.Fatalf("Set(db-proxy) error: %v", err)
	}

	if err := runReplay(cmd, nil); err != nil {
		t.Fatalf("runReplay() error: %v", err)
	}
	if !fakeHook.startInvoked {
		t.Fatal("expected replay db hook to start")
	}
	if !fakeHook.stopped {
		t.Fatal("expected replay db hook to stop")
	}
}

func TestRunReplay_ConfigDBProxyStartsHooks(t *testing.T) {
	withRuntimeConfig(t, func(cfg *config.AppConfig) {
		cfg.Replay.DBProxies = []config.DBProxyConfig{
			{Type: "postgres", ListenAddr: ":15433", TargetAddr: "127.0.0.1:5432"},
		}
	})
	store := newStoreForRuntime(t)
	session := &model.Session{Name: "replay-config-db-proxy", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := store.AppendRecord(session.ID, &model.Record{
		ID:        "orig-1",
		SessionID: session.ID,
		Sequence:  1,
		Request:   model.HTTPRequest{Method: "GET", Path: "/hello"},
	}); err != nil {
		t.Fatalf("AppendRecord() error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	fakeHook := &fakeDBHook{sideEffects: make(chan model.SideEffect)}
	oldFactory := newDBHook
	defer func() { newDBHook = oldFactory }()
	var gotConfig dbhook.Config
	newDBHook = func(cfg dbhook.Config) (dbhook.DBHook, error) {
		gotConfig = cfg
		return fakeHook, nil
	}

	replaySession = session.Name
	replayTarget = server.URL
	replayConcurrency = 1
	replayDelay = ""
	replayDBProxy = nil

	cmd := &cobra.Command{}
	cmd.Flags().IntVar(&replayConcurrency, "concurrency", 1, "")
	cmd.Flags().StringVar(&replayDelay, "delay", "", "")
	cmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "")

	if err := runReplay(cmd, nil); err != nil {
		t.Fatalf("runReplay() error: %v", err)
	}
	if !fakeHook.startInvoked {
		t.Fatal("expected replay db hook to start from config")
	}
	if gotConfig.DBType != "postgres" || gotConfig.ListenAddr != ":15433" {
		t.Fatalf("unexpected hook config: %+v", gotConfig)
	}
	if !fakeHook.stopped {
		t.Fatal("expected replay db hook to stop")
	}
}
