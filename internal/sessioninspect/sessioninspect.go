package sessioninspect

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

// Store is the storage capability needed for session inspection.
type Store interface {
	Get(id string) (*model.Session, error)
	ListRecords(sessionID string) ([]model.Record, error)
	ListReplayRecords(sessionID string) ([]model.Record, error)
	LoadResults(sessionID string) ([]model.DiffResult, error)
}

// Report describes session artifacts and side-effect counts.
type Report struct {
	Session           model.Session         `json:"session"`
	DataDir           string                `json:"dataDir"`
	SessionDir        string                `json:"sessionDir"`
	Files             map[string]FileStatus `json:"files"`
	RecordCount       int                   `json:"recordCount"`
	ReplayRecordCount int                   `json:"replayRecordCount"`
	DiffResultCount   int                   `json:"diffResultCount"`
	RecordSideEffects map[string]int        `json:"recordSideEffects"`
	ReplaySideEffects map[string]int        `json:"replaySideEffects"`
	Warnings          []string              `json:"warnings,omitempty"`
}

// FileStatus describes a known session artifact path.
type FileStatus struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size"`
}

// BuildReport inspects one session without modifying it.
func BuildReport(store Store, dataDir, sessionID string) (Report, error) {
	sess, err := store.Get(sessionID)
	if err != nil {
		return Report{}, fmt.Errorf("failed to get session: %w", err)
	}

	records, err := store.ListRecords(sess.ID)
	if err != nil {
		return Report{}, fmt.Errorf("failed to list records: %w", err)
	}
	replays, err := store.ListReplayRecords(sess.ID)
	if err != nil {
		return Report{}, fmt.Errorf("failed to list replay records: %w", err)
	}
	results, err := store.LoadResults(sess.ID)
	if err != nil {
		return Report{}, fmt.Errorf("failed to load diff results: %w", err)
	}

	sessionDir := filepath.Join(dataDir, "sessions", sess.ID)
	files := map[string]FileStatus{
		"session":       inspectFile(filepath.Join(sessionDir, "session.json")),
		"records":       inspectFile(filepath.Join(sessionDir, "records.jsonl")),
		"replayRecords": inspectFile(filepath.Join(sessionDir, "replay-records.jsonl")),
		"diffResults":   inspectFile(filepath.Join(sessionDir, "diff-results.json")),
	}

	report := Report{
		Session:           *sess,
		DataDir:           dataDir,
		SessionDir:        sessionDir,
		Files:             files,
		RecordCount:       len(records),
		ReplayRecordCount: len(replays),
		DiffResultCount:   len(results),
		RecordSideEffects: CountRecordDBSideEffects(records),
		ReplaySideEffects: CountRecordDBSideEffects(replays),
	}
	if !files["replayRecords"].Exists {
		report.Warnings = append(report.Warnings, "replay records have not been generated")
	}
	if !files["diffResults"].Exists {
		report.Warnings = append(report.Warnings, "diff results have not been generated")
	}
	return report, nil
}

func inspectFile(path string) FileStatus {
	info, err := os.Stat(path)
	if err != nil {
		return FileStatus{Path: path}
	}
	return FileStatus{Path: path, Exists: true, Size: info.Size()}
}

// CountRecordDBSideEffects counts database side effects by supported DB type.
func CountRecordDBSideEffects(records []model.Record) map[string]int {
	counts := map[string]int{}
	for _, typ := range dbtype.Supported() {
		counts[typ] = 0
	}
	counts["unknown"] = 0
	for _, record := range records {
		for _, effect := range record.SideEffects {
			if effect.Type != model.SideEffectDB {
				counts["unknown"]++
				continue
			}
			typ := effect.DatabaseType()
			if typ == "" || !dbtype.IsSupported(typ) {
				counts["unknown"]++
				continue
			}
			counts[typ]++
		}
	}
	return counts
}

// PrintReport writes a terminal session inspection report.
func PrintReport(w io.Writer, report Report) {
	fmt.Fprintf(w, "Session: %s (%s)\n", report.Session.ID, report.Session.Name)
	fmt.Fprintf(w, "Status: %s\n", report.Session.Status)
	fmt.Fprintf(w, "Data: %s\n", report.DataDir)
	fmt.Fprintf(w, "Session dir: %s\n", report.SessionDir)
	fmt.Fprintf(w, "Records: %d\n", report.RecordCount)
	fmt.Fprintf(w, "Replay records: %d\n", report.ReplayRecordCount)
	fmt.Fprintf(w, "Diff results: %d\n", report.DiffResultCount)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Files:")
	fileKeys := sortedFileKeys(report.Files)
	for _, key := range fileKeys {
		status := "missing"
		if report.Files[key].Exists {
			status = fmt.Sprintf("present (%d bytes)", report.Files[key].Size)
		}
		fmt.Fprintf(w, "  %s: %s\n", key, status)
		fmt.Fprintf(w, "    %s\n", report.Files[key].Path)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Record side effects:")
	printCountMap(w, report.RecordSideEffects)
	fmt.Fprintln(w, "Replay side effects:")
	printCountMap(w, report.ReplaySideEffects)

	if len(report.Warnings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Warnings:")
		for _, warning := range report.Warnings {
			fmt.Fprintf(w, "  - %s\n", warning)
		}
	}
}

func printCountMap(w io.Writer, counts map[string]int) {
	for _, key := range sortedIntKeys(counts) {
		fmt.Fprintf(w, "  %s: %d\n", key, counts[key])
	}
}

func sortedFileKeys(m map[string]FileStatus) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedIntKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
