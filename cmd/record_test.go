package cmd

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"shadiff/internal/config"
	"shadiff/internal/daemon"
	"shadiff/internal/model"

	"github.com/spf13/cobra"
)

func TestBuildDaemonChildArgs(t *testing.T) {
	oldCfgFile := cfgFile
	oldRecordTarget := recordTarget
	oldRecordListen := recordListen
	oldRecordDuration := recordDuration
	oldRuntime := runtimeCtx
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		recordTarget = oldRecordTarget
		recordListen = oldRecordListen
		recordDuration = oldRecordDuration
		runtimeCtx = oldRuntime
	})

	cfgFile = "E:/tmp/config.json"
	recordTarget = "http://old-api:8080"
	recordListen = ":19090"
	recordDuration = "5m"
	runtimeCtx = &appRuntime{ConfigPath: cfgFile}

	args := buildDaemonChildArgs("sess-123", []config.DBProxyConfig{
		{Type: "mysql", ListenAddr: ":13306", TargetAddr: "127.0.0.1:3306"},
	})

	expected := []string{
		"record",
		"--target", "http://old-api:8080",
		"--listen", ":19090",
		"--session", "sess-123",
		"--_daemon-child",
		"--config", "E:/tmp/config.json",
		"--duration", "5m",
		"--db-proxy", "mysql://:13306->127.0.0.1:3306",
	}

	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d\nargs=%v", len(args), len(expected), args)
	}
	for i := range expected {
		if args[i] != expected[i] {
			t.Fatalf("args[%d] = %q, want %q\nargs=%v", i, args[i], expected[i], args)
		}
	}
}

func newRecordTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("listen", ":18080", "")
	cmd.Flags().StringArray("db-proxy", nil, "")
	return cmd
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func TestRunRecord_UsesConfigWhenFlagsUnset(t *testing.T) {
	withRuntimeConfig(t, func(cfg *config.AppConfig) {
		cfg.Capture.ListenAddr = "127.0.0.1:19090"
		cfg.Capture.DBProxies = []config.DBProxyConfig{
			{Type: "mysql", ListenAddr: ":13306", TargetAddr: "127.0.0.1:3306"},
		}
	})

	oldRecordTarget := recordTarget
	oldRecordListen := recordListen
	oldRecordDBProxy := recordDBProxy
	oldRecordDaemon := recordDaemon
	oldRunDaemonParentFn := runDaemonParentFn
	oldRunRecordLoopFn := runRecordLoopFn
	t.Cleanup(func() {
		recordTarget = oldRecordTarget
		recordListen = oldRecordListen
		recordDBProxy = oldRecordDBProxy
		recordDaemon = oldRecordDaemon
		runDaemonParentFn = oldRunDaemonParentFn
		runRecordLoopFn = oldRunRecordLoopFn
	})

	recordTarget = "http://example.com"
	recordListen = ":18080"
	recordDBProxy = nil
	recordDaemon = false

	var gotDataDir string
	var gotProxies []config.DBProxyConfig
	runDaemonParentFn = func(*cobra.Command, string, []config.DBProxyConfig) error {
		t.Fatal("runDaemonParentFn should not be called")
		return nil
	}
	runRecordLoopFn = func(dataDir string, dbProxies []config.DBProxyConfig) error {
		gotDataDir = dataDir
		gotProxies = append([]config.DBProxyConfig(nil), dbProxies...)
		return nil
	}

	if err := runRecord(newRecordTestCommand(), nil); err != nil {
		t.Fatalf("runRecord() error: %v", err)
	}

	if recordListen != "127.0.0.1:19090" {
		t.Fatalf("recordListen = %q, want %q", recordListen, "127.0.0.1:19090")
	}
	if gotDataDir != currentDataDir() {
		t.Fatalf("dataDir = %q, want %q", gotDataDir, currentDataDir())
	}
	expected := []config.DBProxyConfig{
		{Type: "mysql", ListenAddr: ":13306", TargetAddr: "127.0.0.1:3306"},
	}
	if !reflect.DeepEqual(gotProxies, expected) {
		t.Fatalf("db proxies = %#v, want %#v", gotProxies, expected)
	}
}

func TestRunRecord_PrefersFlagsOverConfigAndUsesDaemonBranch(t *testing.T) {
	dataDir := withRuntimeConfig(t, func(cfg *config.AppConfig) {
		cfg.Capture.ListenAddr = "127.0.0.1:19090"
		cfg.Capture.DBProxies = []config.DBProxyConfig{
			{Type: "mysql", ListenAddr: ":13306", TargetAddr: "127.0.0.1:3306"},
		}
	})

	oldRecordTarget := recordTarget
	oldRecordListen := recordListen
	oldRecordDBProxy := recordDBProxy
	oldRecordDaemon := recordDaemon
	oldRunDaemonParentFn := runDaemonParentFn
	oldRunRecordLoopFn := runRecordLoopFn
	t.Cleanup(func() {
		recordTarget = oldRecordTarget
		recordListen = oldRecordListen
		recordDBProxy = oldRecordDBProxy
		recordDaemon = oldRecordDaemon
		runDaemonParentFn = oldRunDaemonParentFn
		runRecordLoopFn = oldRunRecordLoopFn
	})

	recordTarget = "http://example.com"
	recordDaemon = true

	cmd := newRecordTestCommand()
	if err := cmd.Flags().Set("listen", "127.0.0.1:29090"); err != nil {
		t.Fatalf("Set(listen) error: %v", err)
	}
	recordListen = "127.0.0.1:29090"
	if err := cmd.Flags().Set("db-proxy", "postgres://:15432->127.0.0.1:5432"); err != nil {
		t.Fatalf("Set(db-proxy) error: %v", err)
	}
	recordDBProxy = []string{"postgres://:15432->127.0.0.1:5432"}

	var gotDataDir string
	var gotProxies []config.DBProxyConfig
	runDaemonParentFn = func(_ *cobra.Command, dataDirArg string, dbProxies []config.DBProxyConfig) error {
		gotDataDir = dataDirArg
		gotProxies = append([]config.DBProxyConfig(nil), dbProxies...)
		return nil
	}
	runRecordLoopFn = func(string, []config.DBProxyConfig) error {
		t.Fatal("runRecordLoopFn should not be called")
		return nil
	}

	if err := runRecord(cmd, nil); err != nil {
		t.Fatalf("runRecord() error: %v", err)
	}

	if gotDataDir != dataDir {
		t.Fatalf("dataDir = %q, want %q", gotDataDir, dataDir)
	}
	expected := []config.DBProxyConfig{
		{Type: "postgres", ListenAddr: ":15432", TargetAddr: "127.0.0.1:5432"},
	}
	if !reflect.DeepEqual(gotProxies, expected) {
		t.Fatalf("db proxies = %#v, want %#v", gotProxies, expected)
	}
}

func TestRunDaemonParent_CreatesSessionAndPIDFile(t *testing.T) {
	dataDir := withRuntimeConfig(t, nil)

	oldCfgFile := cfgFile
	oldRecordTarget := recordTarget
	oldRecordListen := recordListen
	oldRecordSession := recordSession
	oldCurrentExecutable := currentExecutable
	oldExecCommand := execCommand
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		recordTarget = oldRecordTarget
		recordListen = oldRecordListen
		recordSession = oldRecordSession
		currentExecutable = oldCurrentExecutable
		execCommand = oldExecCommand
	})

	cfgFile = currentConfigPath()
	recordTarget = "http://example.com"
	recordListen = "127.0.0.1:19090"
	recordSession = "daemon-case"
	currentExecutable = func() (string, error) { return "ignored", nil }
	execCommand = func(string, ...string) *exec.Cmd {
		if runtime.GOOS == "windows" {
			return exec.Command("cmd", "/c", "exit 0")
		}
		return exec.Command("sh", "-c", "exit 0")
	}

	if err := runDaemonParent(newRecordTestCommand(), dataDir, nil); err != nil {
		t.Fatalf("runDaemonParent() error: %v", err)
	}

	store := newStoreForRuntime(t)
	sessions, err := store.List(nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}

	session := sessions[0]
	if !session.DaemonMode {
		t.Fatal("expected daemon mode session")
	}
	if session.PID == 0 {
		t.Fatal("expected session PID to be populated")
	}

	sessionDir := filepath.Join(dataDir, "sessions", session.ID)
	pid, err := daemon.ReadPID(sessionDir)
	if err != nil {
		t.Fatalf("ReadPID() error: %v", err)
	}
	if pid != session.PID {
		t.Fatalf("pidfile = %d, want %d", pid, session.PID)
	}
	if _, err := os.Stat(filepath.Join(sessionDir, "daemon.log")); err != nil {
		t.Fatalf("expected daemon log to exist: %v", err)
	}
}

func TestRunRecordLoop_RecordsRequestAndCompletesSession(t *testing.T) {
	withRuntimeConfig(t, nil)

	oldRecordTarget := recordTarget
	oldRecordListen := recordListen
	oldRecordSession := recordSession
	oldRecordDuration := recordDuration
	oldRecordDaemon := recordDaemon
	oldDaemonChild := daemonChild
	t.Cleanup(func() {
		recordTarget = oldRecordTarget
		recordListen = oldRecordListen
		recordSession = oldRecordSession
		recordDuration = oldRecordDuration
		recordDaemon = oldRecordDaemon
		daemonChild = oldDaemonChild
	})

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/hello")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	defer target.Close()

	recordTarget = target.URL
	recordListen = freeTCPAddr(t)
	recordSession = "loop-case"
	recordDuration = "1200ms"
	recordDaemon = false
	daemonChild = false

	errCh := make(chan error, 1)
	go func() {
		errCh <- runRecordLoop(currentDataDir(), nil)
	}()

	proxyURL := "http://" + recordListen + "/hello"
	client := &http.Client{Timeout: 200 * time.Millisecond}
	requestSent := false
	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		resp, err := client.Get(proxyURL)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
			}
			requestSent = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !requestSent {
		t.Fatalf("failed to reach proxy at %s before timeout", proxyURL)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("runRecordLoop() error: %v", err)
	}

	store := newStoreForRuntime(t)
	sessions, err := store.List(nil)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	if sessions[0].Status != model.SessionCompleted {
		t.Fatalf("session status = %q, want %q", sessions[0].Status, model.SessionCompleted)
	}
	if sessions[0].RecordCount != 1 {
		t.Fatalf("record count = %d, want 1", sessions[0].RecordCount)
	}

	records, err := store.ListRecords(sessions[0].ID)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].Request.Path != "/hello" {
		t.Fatalf("recorded path = %q, want %q", records[0].Request.Path, "/hello")
	}
}
