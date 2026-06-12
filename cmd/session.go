package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage recording sessions",
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	RunE:  runSessionList,
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show session details",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionShow,
}

var sessionInspectCmd = &cobra.Command{
	Use:   "inspect <id>",
	Short: "Inspect session artifacts and side-effect counts",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionInspect,
}

var sessionDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionDelete,
}

var (
	sessionTagFilter     string
	sessionInspectFormat string
)

func init() {
	sessionListCmd.Flags().StringVar(&sessionTagFilter, "tag", "", "Filter by tag")
	sessionInspectCmd.Flags().StringVar(&sessionInspectFormat, "format", "terminal", "Output format: terminal, json")

	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	sessionCmd.AddCommand(sessionInspectCmd)
	sessionCmd.AddCommand(sessionDeleteCmd)
	rootCmd.AddCommand(sessionCmd)
}

func getStore() (*storage.FileStore, error) {
	return storage.NewFileStore(currentDataDir())
}

func runSessionList(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	var filter *model.SessionFilter
	if sessionTagFilter != "" {
		filter = &model.SessionFilter{Tags: []string{sessionTagFilter}}
	}

	sessions, err := store.List(filter)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tRECORDS\tCREATED")
	for _, s := range sessions {
		created := time.UnixMilli(s.CreatedAt).Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", s.ID, s.Name, s.Status, s.RecordCount, created)
	}
	return w.Flush()
}

func runSessionShow(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	sess, err := store.Get(args[0])
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	fmt.Printf("ID:          %s\n", sess.ID)
	fmt.Printf("Name:        %s\n", sess.Name)
	fmt.Printf("Description: %s\n", sess.Description)
	fmt.Printf("Status:      %s\n", sess.Status)
	fmt.Printf("Records:     %d\n", sess.RecordCount)
	fmt.Printf("Source:      %s\n", sess.Source.BaseURL)
	fmt.Printf("Target:      %s\n", sess.Target.BaseURL)
	fmt.Printf("Tags:        %v\n", sess.Tags)
	fmt.Printf("Created:     %s\n", time.UnixMilli(sess.CreatedAt).Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", time.UnixMilli(sess.UpdatedAt).Format("2006-01-02 15:04:05"))

	return nil
}

type sessionInspectReport struct {
	Session           model.Session         `json:"session"`
	DataDir           string                `json:"dataDir"`
	SessionDir        string                `json:"sessionDir"`
	Files             map[string]fileStatus `json:"files"`
	RecordCount       int                   `json:"recordCount"`
	ReplayRecordCount int                   `json:"replayRecordCount"`
	DiffResultCount   int                   `json:"diffResultCount"`
	RecordSideEffects map[string]int        `json:"recordSideEffects"`
	ReplaySideEffects map[string]int        `json:"replaySideEffects"`
	Warnings          []string              `json:"warnings,omitempty"`
}

type fileStatus struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size"`
}

func runSessionInspect(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	sessionID, err := resolveSession(store, args[0])
	if err != nil {
		return fmt.Errorf("failed to resolve session: %w", err)
	}
	report, err := buildSessionInspectReport(store, sessionID)
	if err != nil {
		return err
	}

	switch sessionInspectFormat {
	case "", "terminal":
		printSessionInspect(cmd.OutOrStdout(), report)
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("write session inspect JSON: %w", err)
		}
	default:
		return fmt.Errorf("invalid session inspect output format %q: must be terminal or json", sessionInspectFormat)
	}
	return nil
}

func buildSessionInspectReport(store *storage.FileStore, sessionID string) (sessionInspectReport, error) {
	sess, err := store.Get(sessionID)
	if err != nil {
		return sessionInspectReport{}, fmt.Errorf("failed to get session: %w", err)
	}

	records, err := store.ListRecords(sess.ID)
	if err != nil {
		return sessionInspectReport{}, fmt.Errorf("failed to list records: %w", err)
	}
	replays, err := store.ListReplayRecords(sess.ID)
	if err != nil {
		return sessionInspectReport{}, fmt.Errorf("failed to list replay records: %w", err)
	}
	results, err := store.LoadResults(sess.ID)
	if err != nil {
		return sessionInspectReport{}, fmt.Errorf("failed to load diff results: %w", err)
	}

	sessionDir := filepath.Join(currentDataDir(), "sessions", sess.ID)
	files := map[string]fileStatus{
		"session":       inspectFile(filepath.Join(sessionDir, "session.json")),
		"records":       inspectFile(filepath.Join(sessionDir, "records.jsonl")),
		"replayRecords": inspectFile(filepath.Join(sessionDir, "replay-records.jsonl")),
		"diffResults":   inspectFile(filepath.Join(sessionDir, "diff-results.json")),
	}

	report := sessionInspectReport{
		Session:           *sess,
		DataDir:           currentDataDir(),
		SessionDir:        sessionDir,
		Files:             files,
		RecordCount:       len(records),
		ReplayRecordCount: len(replays),
		DiffResultCount:   len(results),
		RecordSideEffects: countRecordDBSideEffects(records),
		ReplaySideEffects: countRecordDBSideEffects(replays),
	}
	if !files["replayRecords"].Exists {
		report.Warnings = append(report.Warnings, "replay records have not been generated")
	}
	if !files["diffResults"].Exists {
		report.Warnings = append(report.Warnings, "diff results have not been generated")
	}
	return report, nil
}

func inspectFile(path string) fileStatus {
	info, err := os.Stat(path)
	if err != nil {
		return fileStatus{Path: path}
	}
	return fileStatus{Path: path, Exists: true, Size: info.Size()}
}

func countRecordDBSideEffects(records []model.Record) map[string]int {
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

func printSessionInspect(w io.Writer, report sessionInspectReport) {
	fmt.Fprintf(w, "Session: %s (%s)\n", report.Session.ID, report.Session.Name)
	fmt.Fprintf(w, "Status: %s\n", report.Session.Status)
	fmt.Fprintf(w, "Data: %s\n", report.DataDir)
	fmt.Fprintf(w, "Session dir: %s\n", report.SessionDir)
	fmt.Fprintf(w, "Records: %d\n", report.RecordCount)
	fmt.Fprintf(w, "Replay records: %d\n", report.ReplayRecordCount)
	fmt.Fprintf(w, "Diff results: %d\n", report.DiffResultCount)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Files:")
	fileKeys := sortedKeys(report.Files)
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
	for _, key := range sortedKeysInt(counts) {
		fmt.Fprintf(w, "  %s: %d\n", key, counts[key])
	}
}

func sortedKeys(m map[string]fileStatus) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysInt(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func runSessionDelete(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	// Verify session exists first
	sess, err := store.Get(args[0])
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if err := store.Delete(sess.ID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	fmt.Printf("Session deleted: %s (%s)\n", sess.ID, sess.Name)
	return nil
}
