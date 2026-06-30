package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"shadiff/internal/model"
)

type options struct {
	recordsFile       string
	replayRecordsFile string
	diffFile          string
	reportFile        string
	runID             string
	sessionName       string
	workDir           string
	configFile        string
	artifactsDir      string
	assert            bool
	summaryFile       string
	printSummary      bool
}

type acceptanceSummary struct {
	RunID             string `json:"runId"`
	SessionName       string `json:"sessionName"`
	WorkDir           string `json:"workDir"`
	ConfigFile        string `json:"configFile"`
	ArtifactsDir      string `json:"artifactsDir"`
	RecordsFile       string `json:"recordsFile"`
	ReplayRecordsFile string `json:"replayRecordsFile"`
	DiffFile          string `json:"diffFile"`
	ReportFile        string `json:"reportFile"`
	TotalCount        int    `json:"totalCount"`
	DiffCount         int    `json:"diffCount"`
	HTTPMatch         bool   `json:"httpMatch"`
	HasSQLDiff        bool   `json:"hasSQLDiff"`
	HasMongoDiff      bool   `json:"hasMongoDiff"`
	HasRedisDiff      bool   `json:"hasRedisDiff"`
}

type diffJSON struct {
	Summary struct {
		TotalCount int `json:"totalCount"`
		DiffCount  int `json:"diffCount"`
	} `json:"summary"`
	Results []model.DiffResult `json:"results"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}

	summary, err := buildSummary(opts)
	if err != nil {
		return err
	}

	if opts.summaryFile != "" {
		if err := writeSummary(opts.summaryFile, summary); err != nil {
			return err
		}
	}
	if opts.printSummary {
		if err := printSummary(stdout, summary); err != nil {
			return err
		}
	}
	if opts.assert {
		return assertSummary(summary)
	}
	return nil
}

func parseOptions(args []string) (options, error) {
	var opts options
	fs := flag.NewFlagSet("e2e-assert", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.recordsFile, "records", "", "Path to records.jsonl")
	fs.StringVar(&opts.replayRecordsFile, "replay-records", "", "Path to replay-records.jsonl")
	fs.StringVar(&opts.diffFile, "diff", "", "Path to diff.json")
	fs.StringVar(&opts.reportFile, "report", "", "Path to report.html")
	fs.StringVar(&opts.runID, "run-id", "", "E2E run ID")
	fs.StringVar(&opts.sessionName, "session-name", "", "Session name")
	fs.StringVar(&opts.workDir, "work-dir", "", "Work directory")
	fs.StringVar(&opts.configFile, "config-file", "", "Config file path")
	fs.StringVar(&opts.artifactsDir, "artifacts-dir", "", "Artifacts directory")
	fs.BoolVar(&opts.assert, "assert", false, "Validate expected E2E acceptance conditions")
	fs.StringVar(&opts.summaryFile, "summary-file", "", "Write summary JSON to this path")
	fs.BoolVar(&opts.printSummary, "print-summary", false, "Print summary JSON to stdout")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return opts, nil
}

func buildSummary(opts options) (acceptanceSummary, error) {
	if err := requireNonEmptyFile("records", opts.recordsFile); err != nil {
		return acceptanceSummary{}, err
	}
	if err := requireNonEmptyFile("replay records", opts.replayRecordsFile); err != nil {
		return acceptanceSummary{}, err
	}
	if err := requireNonEmptyFile("diff", opts.diffFile); err != nil {
		return acceptanceSummary{}, err
	}
	if err := requireNonEmptyFile("report", opts.reportFile); err != nil {
		return acceptanceSummary{}, err
	}

	report, err := loadDiffJSON(opts.diffFile)
	if err != nil {
		return acceptanceSummary{}, err
	}

	summary := acceptanceSummary{
		RunID:             opts.runID,
		SessionName:       opts.sessionName,
		WorkDir:           opts.workDir,
		ConfigFile:        opts.configFile,
		ArtifactsDir:      opts.artifactsDir,
		RecordsFile:       opts.recordsFile,
		ReplayRecordsFile: opts.replayRecordsFile,
		DiffFile:          opts.diffFile,
		ReportFile:        opts.reportFile,
		TotalCount:        report.Summary.TotalCount,
		DiffCount:         report.Summary.DiffCount,
		HTTPMatch:         true,
	}

	for _, result := range report.Results {
		for _, diff := range result.Differences {
			switch diff.Kind {
			case model.DiffStatusCode, model.DiffHeader, model.DiffBody, model.DiffBodyField:
				summary.HTTPMatch = false
			case model.DiffDBQuery:
				summary.HasSQLDiff = true
			case model.DiffMongoOp:
				summary.HasMongoDiff = true
			case model.DiffRedisCommand:
				summary.HasRedisDiff = true
			}
		}
	}

	return summary, nil
}

func requireNonEmptyFile(label, path string) error {
	if path == "" {
		return fmt.Errorf("%s file path is empty", label)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s file cannot be inspected: %w", label, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%s file is empty: %s", label, path)
	}
	return nil
}

func loadDiffJSON(path string) (diffJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return diffJSON{}, fmt.Errorf("read diff JSON: %w", err)
	}
	var report diffJSON
	if err := json.Unmarshal(data, &report); err != nil {
		return diffJSON{}, fmt.Errorf("parse diff JSON: %w", err)
	}
	return report, nil
}

func writeSummary(path string, summary acceptanceSummary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create summary directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create summary file: %w", err)
	}
	defer f.Close()
	return encodeSummary(f, summary)
}

func printSummary(w io.Writer, summary acceptanceSummary) error {
	return encodeSummary(w, summary)
}

func encodeSummary(w io.Writer, summary acceptanceSummary) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		return fmt.Errorf("write summary JSON: %w", err)
	}
	return nil
}

func assertSummary(summary acceptanceSummary) error {
	switch {
	case summary.TotalCount < 1:
		return errors.New("diff summary has no records")
	case summary.DiffCount < 1:
		return errors.New("diff summary has no expected differences")
	case !summary.HTTPMatch:
		return errors.New("diff.json contains HTTP response differences")
	case !summary.HasSQLDiff:
		return errors.New("diff.json does not contain a SQL side-effect difference")
	case !summary.HasMongoDiff:
		return errors.New("diff.json does not contain a MongoDB side-effect difference")
	case !summary.HasRedisDiff:
		return errors.New("diff.json does not contain a Redis side-effect difference")
	default:
		return nil
	}
}
