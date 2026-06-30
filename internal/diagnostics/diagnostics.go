package diagnostics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"shadiff/internal/config"
	"shadiff/internal/dbtype"
)

// Status is the result state for a diagnostic check.
type Status string

const (
	Pass  Status = "pass"
	Warn  Status = "warn"
	Error Status = "error"
	Skip  Status = "skip"
)

// Options controls doctor report generation.
type Options struct {
	Strict     bool
	IncludeE2E bool
	ConfigPath string
	Version    string
	Commit     string
	BuildDate  string
	Command    CommandRunner
	E2EAddrs   []string
}

// Summary counts diagnostic check results.
type Summary struct {
	Pass  int  `json:"pass"`
	Warn  int  `json:"warn"`
	Error int  `json:"error"`
	Skip  int  `json:"skip"`
	OK    bool `json:"ok"`
}

// Check is one diagnostic result.
type Check struct {
	ID      string         `json:"id"`
	Status  Status         `json:"status"`
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

// Report is the full doctor report.
type Report struct {
	Version    string  `json:"version"`
	Commit     string  `json:"commit"`
	BuildDate  string  `json:"buildDate"`
	ConfigPath string  `json:"configPath"`
	DataDir    string  `json:"dataDir"`
	LogDir     string  `json:"logDir"`
	Strict     bool    `json:"strict"`
	E2E        bool    `json:"e2e"`
	Summary    Summary `json:"summary"`
	Checks     []Check `json:"checks"`
}

// CommandResult is the outcome of checking an external command.
type CommandResult struct {
	Found  bool
	Output string
	Err    error
}

// CommandRunner runs external tool checks.
type CommandRunner func(context.Context, string, ...string) CommandResult

var defaultE2EAddrs = []string{
	"127.0.0.1:18081",
	"127.0.0.1:18082",
	"127.0.0.1:18080",
	"127.0.0.1:33306",
	"127.0.0.1:35432",
	"127.0.0.1:37017",
	"127.0.0.1:36379",
	"127.0.0.1:13306",
	"127.0.0.1:15432",
	"127.0.0.1:27018",
	"127.0.0.1:16379",
	"127.0.0.1:13316",
	"127.0.0.1:15442",
	"127.0.0.1:27028",
	"127.0.0.1:16389",
}

// DefaultE2EAddrs returns the default official E2E demo listen addresses.
func DefaultE2EAddrs() []string {
	return append([]string(nil), defaultE2EAddrs...)
}

// BuildReport creates a read-only diagnostic report.
func BuildReport(ctx context.Context, opts Options) (Report, error) {
	configPath := opts.ConfigPath
	if configPath == "" {
		var err error
		configPath, err = config.DefaultPath()
		if err != nil {
			return Report{}, fmt.Errorf("resolve default config path: %w", err)
		}
	}

	cfg, configExists, configLoadErr := loadConfig(configPath)
	dataDir := dataDir(cfg)
	logDir := cfg.Log.LogDir
	if logDir == "" {
		logDir = filepath.Join(dataDir, "logs")
	}

	report := Report{
		Version:    opts.Version,
		Commit:     opts.Commit,
		BuildDate:  opts.BuildDate,
		ConfigPath: configPath,
		DataDir:    dataDir,
		LogDir:     logDir,
		Strict:     opts.Strict,
		E2E:        opts.IncludeE2E,
	}

	report.addCheck(Check{
		ID:      "version",
		Status:  Pass,
		Message: fmt.Sprintf("shadiff %s", opts.Version),
		Detail:  map[string]any{"commit": opts.Commit, "buildDate": opts.BuildDate},
	})

	if configExists {
		report.addCheck(Check{ID: "config.path", Status: Pass, Message: "Config file found", Detail: map[string]any{"path": configPath}})
	} else {
		report.addCheck(Check{ID: "config.path", Status: Warn, Message: "Config file not found; defaults will be used", Detail: map[string]any{"path": configPath}})
	}

	if configLoadErr != nil {
		report.addCheck(Check{ID: "config.valid", Status: Error, Message: "Config file could not be loaded", Detail: map[string]any{"error": configLoadErr.Error()}})
	} else if err := config.Validate(cfg); err != nil {
		report.addCheck(Check{ID: "config.valid", Status: Error, Message: "Config validation failed", Detail: map[string]any{"error": err.Error()}})
	} else {
		report.addCheck(Check{ID: "config.valid", Status: Pass, Message: "Config is valid"})
	}

	report.addCheck(checkPathStatus("storage.dataDir", dataDir, true))
	report.addCheck(checkPathStatus("log.logDir", logDir, true))
	report.addCheck(Check{
		ID:      "dbproxy.supportedTypes",
		Status:  Pass,
		Message: "Supported DB proxy types are registered",
		Detail:  map[string]any{"types": dbtype.Supported()},
	})
	report.addCheck(checkConfigDBProxies(cfg))

	runner := opts.Command
	if runner == nil {
		runner = RunExternalCommand
	}
	report.addCheck(checkExternalTool(ctx, runner, "docker", "version", "--format", "{{.Server.Version}}"))
	report.addCheck(checkDockerCompose(ctx, runner))

	if opts.IncludeE2E {
		addrs := opts.E2EAddrs
		if len(addrs) == 0 {
			addrs = defaultE2EAddrs
		}
		for _, addr := range addrs {
			report.addCheck(checkListenAddrAvailable(addr))
		}
	}

	report.Summary.OK = report.Summary.Error == 0 && (!opts.Strict || report.Summary.Warn == 0)
	return report, nil
}

func loadConfig(path string) (*config.AppConfig, bool, error) {
	cfg := config.DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, false, nil
		}
		return cfg, true, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, true, err
	}
	return cfg, true, nil
}

func dataDir(cfg *config.AppConfig) string {
	if cfg != nil && cfg.Storage.DataDir != "" {
		return cfg.Storage.DataDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".shadiff"
	}
	return filepath.Join(homeDir, ".shadiff")
}

func (r *Report) addCheck(check Check) {
	r.Checks = append(r.Checks, check)
	switch check.Status {
	case Pass:
		r.Summary.Pass++
	case Warn:
		r.Summary.Warn++
	case Error:
		r.Summary.Error++
	case Skip:
		r.Summary.Skip++
	}
}

func checkPathStatus(id, path string, expectDir bool) Check {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Check{ID: id, Status: Warn, Message: "Path does not exist yet", Detail: map[string]any{"path": path}}
		}
		return Check{ID: id, Status: Error, Message: "Path cannot be inspected", Detail: map[string]any{"path": path, "error": err.Error()}}
	}
	if expectDir && !info.IsDir() {
		return Check{ID: id, Status: Error, Message: "Path exists but is not a directory", Detail: map[string]any{"path": path}}
	}
	return Check{ID: id, Status: Pass, Message: "Path is available", Detail: map[string]any{"path": path}}
}

func checkConfigDBProxies(cfg *config.AppConfig) Check {
	if cfg == nil {
		return Check{ID: "dbproxy.config", Status: Skip, Message: "Config is unavailable"}
	}
	captureCount := len(cfg.Capture.DBProxies)
	replayCount := len(cfg.Replay.DBProxies)
	if captureCount == 0 && replayCount == 0 {
		return Check{ID: "dbproxy.config", Status: Pass, Message: "No DB proxies configured", Detail: map[string]any{"capture": 0, "replay": 0}}
	}
	for i, proxy := range cfg.Capture.DBProxies {
		if !dbtype.IsSupported(proxy.Type) {
			return Check{ID: "dbproxy.config", Status: Error, Message: "Unsupported capture DB proxy type", Detail: map[string]any{"index": i, "type": proxy.Type}}
		}
	}
	for i, proxy := range cfg.Replay.DBProxies {
		if !dbtype.IsSupported(proxy.Type) {
			return Check{ID: "dbproxy.config", Status: Error, Message: "Unsupported replay DB proxy type", Detail: map[string]any{"index": i, "type": proxy.Type}}
		}
	}
	return Check{
		ID:      "dbproxy.config",
		Status:  Pass,
		Message: "Configured DB proxy types are supported",
		Detail:  map[string]any{"capture": captureCount, "replay": replayCount},
	}
}

func checkExternalTool(ctx context.Context, runner CommandRunner, name string, args ...string) Check {
	result := runner(ctx, name, args...)
	id := "tool." + name
	if !result.Found {
		return Check{ID: id, Status: Warn, Message: name + " was not found on PATH"}
	}
	if result.Err != nil {
		return Check{ID: id, Status: Warn, Message: name + " check failed", Detail: map[string]any{"error": result.Err.Error(), "output": strings.TrimSpace(result.Output)}}
	}
	return Check{ID: id, Status: Pass, Message: name + " is available", Detail: map[string]any{"output": strings.TrimSpace(result.Output)}}
}

func checkDockerCompose(ctx context.Context, runner CommandRunner) Check {
	result := runner(ctx, "docker", "compose", "version", "--short")
	if result.Found && result.Err == nil {
		return Check{ID: "tool.dockerCompose", Status: Pass, Message: "Docker Compose plugin is available", Detail: map[string]any{"output": strings.TrimSpace(result.Output)}}
	}

	legacy := runner(ctx, "docker-compose", "version", "--short")
	if legacy.Found && legacy.Err == nil {
		return Check{ID: "tool.dockerCompose", Status: Pass, Message: "docker-compose is available", Detail: map[string]any{"output": strings.TrimSpace(legacy.Output)}}
	}

	if !result.Found && !legacy.Found {
		return Check{ID: "tool.dockerCompose", Status: Warn, Message: "Docker Compose was not found on PATH"}
	}
	if result.Err != nil {
		return Check{ID: "tool.dockerCompose", Status: Warn, Message: "Docker Compose check failed", Detail: map[string]any{"error": result.Err.Error(), "output": strings.TrimSpace(result.Output)}}
	}
	return Check{ID: "tool.dockerCompose", Status: Warn, Message: "Docker Compose check failed", Detail: map[string]any{"error": legacy.Err.Error(), "output": strings.TrimSpace(legacy.Output)}}
}

// RunExternalCommand runs an external tool check with a short timeout.
func RunExternalCommand(ctx context.Context, name string, args ...string) CommandResult {
	if _, err := exec.LookPath(name); err != nil {
		return CommandResult{Found: false, Err: err}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return CommandResult{Found: true, Output: string(output), Err: err}
}

func checkListenAddrAvailable(addr string) Check {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return Check{ID: "e2e.port." + strings.ReplaceAll(addr, ":", "_"), Status: Error, Message: "E2E port is not available", Detail: map[string]any{"addr": addr, "error": err.Error()}}
	}
	_ = ln.Close()
	return Check{ID: "e2e.port." + strings.ReplaceAll(addr, ":", "_"), Status: Pass, Message: "E2E port is available", Detail: map[string]any{"addr": addr}}
}

// PrintReport writes the terminal doctor report.
func PrintReport(w interface{ Write([]byte) (int, error) }, report Report) {
	fmt.Fprintln(w, "Shadiff Doctor")
	fmt.Fprintf(w, "Version: %s\n", report.Version)
	fmt.Fprintf(w, "Commit: %s\n", report.Commit)
	fmt.Fprintf(w, "Built: %s\n", report.BuildDate)
	fmt.Fprintf(w, "Config: %s\n", report.ConfigPath)
	fmt.Fprintf(w, "Data: %s\n", report.DataDir)
	fmt.Fprintf(w, "Logs: %s\n", report.LogDir)
	fmt.Fprintln(w)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tCHECK\tMESSAGE")
	for _, check := range report.Checks {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", strings.ToUpper(string(check.Status)), check.ID, check.Message)
	}
	_ = tw.Flush()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Summary: %d passed, %d warning(s), %d error(s), %d skipped\n",
		report.Summary.Pass, report.Summary.Warn, report.Summary.Error, report.Summary.Skip)
}
