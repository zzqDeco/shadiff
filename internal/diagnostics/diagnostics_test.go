package diagnostics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shadiff/internal/config"
)

func TestBuildReport_MissingConfigIsReadOnlyWarning(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing", "config.json")
	report, err := BuildReport(context.Background(), Options{
		ConfigPath: configPath,
		Command:    stubCommand(CommandResult{Found: true, Output: "ok\n"}),
	})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("config path stat error = %v, want not exist", err)
	}
	if status := findCheck(t, report, "config.path").Status; status != Warn {
		t.Fatalf("config.path status = %q, want %q", status, Warn)
	}
}

func TestBuildReport_InvalidConfigIsError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"replay":{"concurrency":0}}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report, err := BuildReport(context.Background(), Options{
		ConfigPath: configPath,
		Command:    stubCommand(CommandResult{Found: true, Output: "ok\n"}),
	})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if status := findCheck(t, report, "config.valid").Status; status != Error {
		t.Fatalf("config.valid status = %q, want %q", status, Error)
	}
	if report.Summary.OK {
		t.Fatal("Summary.OK = true, want false")
	}
}

func TestBuildReport_E2EPortCheck(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer ln.Close()

	report, err := BuildReport(context.Background(), Options{
		ConfigPath: filepath.Join(t.TempDir(), "missing.json"),
		IncludeE2E: true,
		Command:    stubCommand(CommandResult{Found: true, Output: "ok\n"}),
		E2EAddrs:   []string{ln.Addr().String()},
	})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	var found bool
	for _, check := range report.Checks {
		if strings.HasPrefix(check.ID, "e2e.port.") {
			found = true
			if check.Status != Error {
				t.Fatalf("E2E port status = %q, want %q", check.Status, Error)
			}
		}
	}
	if !found {
		t.Fatal("missing E2E port check")
	}
}

func TestPrintReportIncludesSummary(t *testing.T) {
	configPath := writeConfig(t, t.TempDir())
	report, err := BuildReport(context.Background(), Options{
		ConfigPath: configPath,
		Version:    "test-version",
		Command:    stubCommand(CommandResult{Found: true, Output: "ok\n"}),
	})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}

	var out bytes.Buffer
	PrintReport(&out, report)
	for _, want := range []string{"Shadiff Doctor", "Version: test-version", "Summary:"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func stubCommand(result CommandResult) CommandRunner {
	return func(context.Context, string, ...string) CommandResult {
		return result
	}
}

func findCheck(t *testing.T, report Report, id string) Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("missing check %q", id)
	return Check{}
}

func writeConfig(t *testing.T, dataDir string) string {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = dataDir
	cfg.Log.LogDir = filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(cfg.Log.LogDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
