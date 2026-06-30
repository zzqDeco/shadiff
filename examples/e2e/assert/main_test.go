package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shadiff/internal/model"
)

func TestBuildSummaryDetectsExpectedSideEffectDiffs(t *testing.T) {
	files := writeAcceptanceFiles(t, []model.Difference{
		{Kind: model.DiffDBQuery},
		{Kind: model.DiffMongoOp},
		{Kind: model.DiffRedisCommand},
	})

	summary, err := buildSummary(files)
	if err != nil {
		t.Fatalf("buildSummary() error = %v", err)
	}
	if summary.TotalCount != 1 || summary.DiffCount != 1 {
		t.Fatalf("summary counts = %+v, want total=1 diff=1", summary)
	}
	if !summary.HTTPMatch || !summary.HasSQLDiff || !summary.HasMongoDiff || !summary.HasRedisDiff {
		t.Fatalf("summary flags = %+v, want HTTP match and all DB diffs", summary)
	}
	if err := assertSummary(summary); err != nil {
		t.Fatalf("assertSummary() error = %v", err)
	}
}

func TestAssertSummaryFailsOnHTTPDiff(t *testing.T) {
	files := writeAcceptanceFiles(t, []model.Difference{
		{Kind: model.DiffBody},
		{Kind: model.DiffDBQuery},
		{Kind: model.DiffMongoOp},
		{Kind: model.DiffRedisCommand},
	})

	summary, err := buildSummary(files)
	if err != nil {
		t.Fatalf("buildSummary() error = %v", err)
	}
	if err := assertSummary(summary); err == nil || !strings.Contains(err.Error(), "HTTP response differences") {
		t.Fatalf("assertSummary() error = %v, want HTTP diff failure", err)
	}
}

func TestAssertSummaryFailsWhenExpectedDBDiffMissing(t *testing.T) {
	files := writeAcceptanceFiles(t, []model.Difference{
		{Kind: model.DiffDBQuery},
		{Kind: model.DiffMongoOp},
	})

	summary, err := buildSummary(files)
	if err != nil {
		t.Fatalf("buildSummary() error = %v", err)
	}
	if err := assertSummary(summary); err == nil || !strings.Contains(err.Error(), "Redis") {
		t.Fatalf("assertSummary() error = %v, want Redis diff failure", err)
	}
}

func TestRunWritesAndPrintsSummary(t *testing.T) {
	files := writeAcceptanceFiles(t, []model.Difference{
		{Kind: model.DiffDBQuery},
		{Kind: model.DiffMongoOp},
		{Kind: model.DiffRedisCommand},
	})
	summaryFile := filepath.Join(t.TempDir(), "summary.json")

	var out bytes.Buffer
	err := run([]string{
		"--records", files.recordsFile,
		"--replay-records", files.replayRecordsFile,
		"--diff", files.diffFile,
		"--report", files.reportFile,
		"--run-id", "run-1",
		"--session-name", "e2e-run-1",
		"--work-dir", files.workDir,
		"--config-file", files.configFile,
		"--artifacts-dir", files.artifactsDir,
		"--summary-file", summaryFile,
		"--print-summary",
		"--assert",
	}, &out)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(out.String(), `"hasRedisDiff": true`) {
		t.Fatalf("printed summary missing redis flag:\n%s", out.String())
	}
	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("ReadFile(summary) error = %v", err)
	}
	if !strings.Contains(string(data), `"sessionName": "e2e-run-1"`) {
		t.Fatalf("summary file = %s", data)
	}
}

func writeAcceptanceFiles(t *testing.T, diffs []model.Difference) options {
	t.Helper()
	workDir := t.TempDir()
	artifactsDir := filepath.Join(workDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	recordsFile := writeFile(t, filepath.Join(workDir, "records.jsonl"), []byte("{}\n"))
	replayFile := writeFile(t, filepath.Join(workDir, "replay-records.jsonl"), []byte("{}\n"))
	reportFile := writeFile(t, filepath.Join(artifactsDir, "report.html"), []byte("<html></html>"))
	configFile := writeFile(t, filepath.Join(workDir, "config.json"), []byte("{}"))
	diffFile := filepath.Join(artifactsDir, "diff.json")

	payload := map[string]any{
		"summary": map[string]any{
			"totalCount": 1,
			"diffCount":  1,
		},
		"results": []model.DiffResult{
			{RecordID: "record-1", Sequence: 1, Differences: diffs},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal(diff) error = %v", err)
	}
	writeFile(t, diffFile, data)

	return options{
		recordsFile:       recordsFile,
		replayRecordsFile: replayFile,
		diffFile:          diffFile,
		reportFile:        reportFile,
		runID:             "run-1",
		sessionName:       "e2e-run-1",
		workDir:           workDir,
		configFile:        configFile,
		artifactsDir:      artifactsDir,
	}
}

func writeFile(t *testing.T, path string, data []byte) string {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	return path
}
