package cmd

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
	"shadiff/internal/diagnostics"

	"github.com/spf13/cobra"
)

func withDoctorCommandStub(t *testing.T, stub diagnostics.CommandRunner) {
	t.Helper()
	old := doctorCommandRunner
	t.Cleanup(func() { doctorCommandRunner = old })
	doctorCommandRunner = stub
}

func TestBuildDoctorReport_MissingConfigIsReadOnlyWarning(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing", "config.json")
	stub := func(context.Context, string, ...string) diagnostics.CommandResult {
		return diagnostics.CommandResult{Found: true, Output: "ok\n"}
	}

	report, err := diagnostics.BuildReport(context.Background(), diagnostics.Options{ConfigPath: configPath, Command: stub})
	if err != nil {
		t.Fatalf("buildDoctorReport() error = %v", err)
	}
	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("config path stat error = %v, want not exist", err)
	}
	if status := findDoctorCheck(t, report, "config.path").Status; status != diagnostics.Warn {
		t.Fatalf("config.path status = %q, want %q", status, diagnostics.Warn)
	}
}

func TestBuildDoctorReport_InvalidConfigIsError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"replay":{"concurrency":0}}`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	stub := func(context.Context, string, ...string) diagnostics.CommandResult {
		return diagnostics.CommandResult{Found: true, Output: "ok\n"}
	}

	report, err := diagnostics.BuildReport(context.Background(), diagnostics.Options{ConfigPath: configPath, Command: stub})
	if err != nil {
		t.Fatalf("buildDoctorReport() error = %v", err)
	}
	if status := findDoctorCheck(t, report, "config.valid").Status; status != diagnostics.Error {
		t.Fatalf("config.valid status = %q, want %q", status, diagnostics.Error)
	}
	if report.Summary.OK {
		t.Fatal("Summary.OK = true, want false")
	}
}

func TestRunDoctor_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	configPath := writeDoctorConfig(t, dir, nil)
	withDoctorCommandStub(t, func(context.Context, string, ...string) diagnostics.CommandResult {
		return diagnostics.CommandResult{Found: true, Output: "ok\n"}
	})

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	oldCfgFile := cfgFile
	oldFormat := doctorFormat
	oldStrict := doctorStrict
	oldE2E := doctorE2E
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		doctorFormat = oldFormat
		doctorStrict = oldStrict
		doctorE2E = oldE2E
	})
	cfgFile = configPath
	doctorFormat = "json"
	doctorStrict = false
	doctorE2E = false

	if err := runDoctor(cmd, nil); err != nil {
		t.Fatalf("runDoctor() error = %v", err)
	}
	var report diagnostics.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("Unmarshal() error = %v\noutput=%s", err, out.String())
	}
	if report.ConfigPath != configPath {
		t.Fatalf("ConfigPath = %q, want %q", report.ConfigPath, configPath)
	}
	if report.Summary.Error != 0 {
		t.Fatalf("Summary.Error = %d, want 0", report.Summary.Error)
	}
}

func TestRunDoctor_StrictFailsOnWarnings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.json")
	withDoctorCommandStub(t, func(context.Context, string, ...string) diagnostics.CommandResult {
		return diagnostics.CommandResult{Found: false, Err: errors.New("not found")}
	})

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	oldCfgFile := cfgFile
	oldFormat := doctorFormat
	oldStrict := doctorStrict
	oldE2E := doctorE2E
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		doctorFormat = oldFormat
		doctorStrict = oldStrict
		doctorE2E = oldE2E
	})
	cfgFile = configPath
	doctorFormat = "terminal"
	doctorStrict = true
	doctorE2E = false

	err := runDoctor(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "warning") {
		t.Fatalf("runDoctor() error = %v, want strict warning failure", err)
	}
}

func TestBuildDoctorReport_E2EPortCheck(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer ln.Close()

	oldAddrs := doctorE2EAddrs
	t.Cleanup(func() { doctorE2EAddrs = oldAddrs })
	doctorE2EAddrs = []string{ln.Addr().String()}
	stub := func(context.Context, string, ...string) diagnostics.CommandResult {
		return diagnostics.CommandResult{Found: true, Output: "ok\n"}
	}

	report, err := diagnostics.BuildReport(context.Background(), diagnostics.Options{
		ConfigPath: filepath.Join(t.TempDir(), "missing.json"),
		IncludeE2E: true,
		Command:    stub,
		E2EAddrs:   doctorE2EAddrs,
	})
	if err != nil {
		t.Fatalf("buildDoctorReport() error = %v", err)
	}
	var found bool
	for _, check := range report.Checks {
		if strings.HasPrefix(check.ID, "e2e.port.") {
			found = true
			if check.Status != diagnostics.Error {
				t.Fatalf("E2E port status = %q, want %q", check.Status, diagnostics.Error)
			}
		}
	}
	if !found {
		t.Fatal("missing E2E port check")
	}
}

func findDoctorCheck(t *testing.T, report diagnostics.Report, id string) diagnostics.Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("missing doctor check %q", id)
	return diagnostics.Check{}
}

func writeDoctorConfig(t *testing.T, dataDir string, mutate func(*config.AppConfig)) string {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = dataDir
	cfg.Log.LogDir = filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(cfg.Log.LogDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if mutate != nil {
		mutate(cfg)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
