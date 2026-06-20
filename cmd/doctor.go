package cmd

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

	"github.com/spf13/cobra"
)

type doctorStatus string

const (
	doctorPass  doctorStatus = "pass"
	doctorWarn  doctorStatus = "warn"
	doctorError doctorStatus = "error"
	doctorSkip  doctorStatus = "skip"
)

type doctorOptions struct {
	Format     string
	Strict     bool
	IncludeE2E bool
	ConfigPath string
}

type doctorSummary struct {
	Pass  int  `json:"pass"`
	Warn  int  `json:"warn"`
	Error int  `json:"error"`
	Skip  int  `json:"skip"`
	OK    bool `json:"ok"`
}

type doctorCheck struct {
	ID      string         `json:"id"`
	Status  doctorStatus   `json:"status"`
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

type doctorReport struct {
	Version    string        `json:"version"`
	Commit     string        `json:"commit"`
	BuildDate  string        `json:"buildDate"`
	ConfigPath string        `json:"configPath"`
	DataDir    string        `json:"dataDir"`
	LogDir     string        `json:"logDir"`
	Strict     bool          `json:"strict"`
	E2E        bool          `json:"e2e"`
	Summary    doctorSummary `json:"summary"`
	Checks     []doctorCheck `json:"checks"`
}

type doctorCommandResult struct {
	Found  bool
	Output string
	Err    error
}

var (
	doctorFormat string
	doctorStrict bool
	doctorE2E    bool

	doctorExecCommand = runDoctorExternalCommand
	doctorE2EAddrs    = []string{
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
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Shadiff environment readiness",
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().StringVar(&doctorFormat, "format", "terminal", "Output format: terminal, json")
	doctorCmd.Flags().BoolVar(&doctorStrict, "strict", false, "Treat warnings as failures")
	doctorCmd.Flags().BoolVar(&doctorE2E, "e2e", false, "Include official E2E demo port checks")
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	opts := doctorOptions{
		Format:     strings.ToLower(doctorFormat),
		Strict:     doctorStrict,
		IncludeE2E: doctorE2E,
		ConfigPath: cfgFile,
	}
	report, err := buildDoctorReport(cmd.Context(), opts)
	if err != nil {
		return err
	}

	switch opts.Format {
	case "", "terminal":
		printDoctorReport(cmd.OutOrStdout(), report)
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("write doctor JSON: %w", err)
		}
	default:
		return fmt.Errorf("invalid doctor output format %q: must be terminal or json", doctorFormat)
	}

	if report.Summary.Error > 0 {
		return fmt.Errorf("doctor found %d error check(s)", report.Summary.Error)
	}
	if report.Strict && report.Summary.Warn > 0 {
		return fmt.Errorf("doctor found %d warning check(s) in strict mode", report.Summary.Warn)
	}
	return nil
}

func buildDoctorReport(ctx context.Context, opts doctorOptions) (doctorReport, error) {
	configPath := opts.ConfigPath
	if configPath == "" {
		var err error
		configPath, err = config.DefaultPath()
		if err != nil {
			return doctorReport{}, fmt.Errorf("resolve default config path: %w", err)
		}
	}

	cfg, configExists, configLoadErr := loadDoctorConfig(configPath)
	dataDir := doctorDataDir(cfg)
	logDir := cfg.Log.LogDir
	if logDir == "" {
		logDir = filepath.Join(dataDir, "logs")
	}

	report := doctorReport{
		Version:    Version,
		Commit:     Commit,
		BuildDate:  BuildDate,
		ConfigPath: configPath,
		DataDir:    dataDir,
		LogDir:     logDir,
		Strict:     opts.Strict,
		E2E:        opts.IncludeE2E,
	}

	report.addCheck(doctorCheck{
		ID:      "version",
		Status:  doctorPass,
		Message: fmt.Sprintf("shadiff %s", Version),
		Detail:  map[string]any{"commit": Commit, "buildDate": BuildDate},
	})

	if configExists {
		report.addCheck(doctorCheck{ID: "config.path", Status: doctorPass, Message: "Config file found", Detail: map[string]any{"path": configPath}})
	} else {
		report.addCheck(doctorCheck{ID: "config.path", Status: doctorWarn, Message: "Config file not found; defaults will be used", Detail: map[string]any{"path": configPath}})
	}

	if configLoadErr != nil {
		report.addCheck(doctorCheck{ID: "config.valid", Status: doctorError, Message: "Config file could not be loaded", Detail: map[string]any{"error": configLoadErr.Error()}})
	} else if err := config.Validate(cfg); err != nil {
		report.addCheck(doctorCheck{ID: "config.valid", Status: doctorError, Message: "Config validation failed", Detail: map[string]any{"error": err.Error()}})
	} else {
		report.addCheck(doctorCheck{ID: "config.valid", Status: doctorPass, Message: "Config is valid"})
	}

	report.addCheck(checkPathStatus("storage.dataDir", dataDir, true))
	report.addCheck(checkPathStatus("log.logDir", logDir, true))
	report.addCheck(doctorCheck{
		ID:      "dbproxy.supportedTypes",
		Status:  doctorPass,
		Message: "Supported DB proxy types are registered",
		Detail:  map[string]any{"types": dbtype.Supported()},
	})
	report.addCheck(checkConfigDBProxies(cfg))
	report.addCheck(checkExternalTool(ctx, "docker", "version", "--format", "{{.Server.Version}}"))
	report.addCheck(checkDockerCompose(ctx))

	if opts.IncludeE2E {
		for _, addr := range doctorE2EAddrs {
			report.addCheck(checkListenAddrAvailable(addr))
		}
	}

	report.Summary.OK = report.Summary.Error == 0 && (!opts.Strict || report.Summary.Warn == 0)
	return report, nil
}

func loadDoctorConfig(path string) (*config.AppConfig, bool, error) {
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

func doctorDataDir(cfg *config.AppConfig) string {
	if cfg != nil && cfg.Storage.DataDir != "" {
		return cfg.Storage.DataDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".shadiff"
	}
	return filepath.Join(homeDir, ".shadiff")
}

func (r *doctorReport) addCheck(check doctorCheck) {
	r.Checks = append(r.Checks, check)
	switch check.Status {
	case doctorPass:
		r.Summary.Pass++
	case doctorWarn:
		r.Summary.Warn++
	case doctorError:
		r.Summary.Error++
	case doctorSkip:
		r.Summary.Skip++
	}
}

func checkPathStatus(id, path string, expectDir bool) doctorCheck {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{ID: id, Status: doctorWarn, Message: "Path does not exist yet", Detail: map[string]any{"path": path}}
		}
		return doctorCheck{ID: id, Status: doctorError, Message: "Path cannot be inspected", Detail: map[string]any{"path": path, "error": err.Error()}}
	}
	if expectDir && !info.IsDir() {
		return doctorCheck{ID: id, Status: doctorError, Message: "Path exists but is not a directory", Detail: map[string]any{"path": path}}
	}
	return doctorCheck{ID: id, Status: doctorPass, Message: "Path is available", Detail: map[string]any{"path": path}}
}

func checkConfigDBProxies(cfg *config.AppConfig) doctorCheck {
	if cfg == nil {
		return doctorCheck{ID: "dbproxy.config", Status: doctorSkip, Message: "Config is unavailable"}
	}
	captureCount := len(cfg.Capture.DBProxies)
	replayCount := len(cfg.Replay.DBProxies)
	if captureCount == 0 && replayCount == 0 {
		return doctorCheck{ID: "dbproxy.config", Status: doctorPass, Message: "No DB proxies configured", Detail: map[string]any{"capture": 0, "replay": 0}}
	}
	for i, proxy := range cfg.Capture.DBProxies {
		if !dbtype.IsSupported(proxy.Type) {
			return doctorCheck{ID: "dbproxy.config", Status: doctorError, Message: "Unsupported capture DB proxy type", Detail: map[string]any{"index": i, "type": proxy.Type}}
		}
	}
	for i, proxy := range cfg.Replay.DBProxies {
		if !dbtype.IsSupported(proxy.Type) {
			return doctorCheck{ID: "dbproxy.config", Status: doctorError, Message: "Unsupported replay DB proxy type", Detail: map[string]any{"index": i, "type": proxy.Type}}
		}
	}
	return doctorCheck{
		ID:      "dbproxy.config",
		Status:  doctorPass,
		Message: "Configured DB proxy types are supported",
		Detail:  map[string]any{"capture": captureCount, "replay": replayCount},
	}
}

func checkExternalTool(ctx context.Context, name string, args ...string) doctorCheck {
	result := doctorExecCommand(ctx, name, args...)
	id := "tool." + name
	if !result.Found {
		return doctorCheck{ID: id, Status: doctorWarn, Message: name + " was not found on PATH"}
	}
	if result.Err != nil {
		return doctorCheck{ID: id, Status: doctorWarn, Message: name + " check failed", Detail: map[string]any{"error": result.Err.Error(), "output": strings.TrimSpace(result.Output)}}
	}
	return doctorCheck{ID: id, Status: doctorPass, Message: name + " is available", Detail: map[string]any{"output": strings.TrimSpace(result.Output)}}
}

func checkDockerCompose(ctx context.Context) doctorCheck {
	result := doctorExecCommand(ctx, "docker", "compose", "version", "--short")
	if result.Found && result.Err == nil {
		return doctorCheck{ID: "tool.dockerCompose", Status: doctorPass, Message: "Docker Compose plugin is available", Detail: map[string]any{"output": strings.TrimSpace(result.Output)}}
	}

	legacy := doctorExecCommand(ctx, "docker-compose", "version", "--short")
	if legacy.Found && legacy.Err == nil {
		return doctorCheck{ID: "tool.dockerCompose", Status: doctorPass, Message: "docker-compose is available", Detail: map[string]any{"output": strings.TrimSpace(legacy.Output)}}
	}

	if !result.Found && !legacy.Found {
		return doctorCheck{ID: "tool.dockerCompose", Status: doctorWarn, Message: "Docker Compose was not found on PATH"}
	}
	if result.Err != nil {
		return doctorCheck{ID: "tool.dockerCompose", Status: doctorWarn, Message: "Docker Compose check failed", Detail: map[string]any{"error": result.Err.Error(), "output": strings.TrimSpace(result.Output)}}
	}
	return doctorCheck{ID: "tool.dockerCompose", Status: doctorWarn, Message: "Docker Compose check failed", Detail: map[string]any{"error": legacy.Err.Error(), "output": strings.TrimSpace(legacy.Output)}}
}

func runDoctorExternalCommand(ctx context.Context, name string, args ...string) doctorCommandResult {
	if _, err := exec.LookPath(name); err != nil {
		return doctorCommandResult{Found: false, Err: err}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return doctorCommandResult{Found: true, Output: string(output), Err: err}
}

func checkListenAddrAvailable(addr string) doctorCheck {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return doctorCheck{ID: "e2e.port." + strings.ReplaceAll(addr, ":", "_"), Status: doctorError, Message: "E2E port is not available", Detail: map[string]any{"addr": addr, "error": err.Error()}}
	}
	_ = ln.Close()
	return doctorCheck{ID: "e2e.port." + strings.ReplaceAll(addr, ":", "_"), Status: doctorPass, Message: "E2E port is available", Detail: map[string]any{"addr": addr}}
}

func printDoctorReport(w anyWriter, report doctorReport) {
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

type anyWriter interface {
	Write([]byte) (int, error)
}
