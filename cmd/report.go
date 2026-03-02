package cmd

import (
	"fmt"
	"os"

	"shadiff/internal/diff"
	"shadiff/internal/logger"
	"shadiff/internal/reporter"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var (
	reportSession string
	reportFormat  string
	reportOutput  string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a detailed report",
	Long: `Generate a detailed report from diff results, supporting terminal/JSON/HTML formats.

Examples:
  shadiff report -s abc123
  shadiff report -s abc123 -f html -o report.html
  shadiff report -s abc123 -f json -o result.json`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().StringVarP(&reportSession, "session", "s", "", "Session ID or name (required)")
	reportCmd.Flags().StringVarP(&reportFormat, "format", "f", "terminal", "Report format: terminal, json, html")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "Output path (default stdout)")

	reportCmd.MarkFlagRequired("session")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/.shadiff"
	if err := logger.Init(dataDir); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()

	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	sessionID, err := resolveSession(store, reportSession)
	if err != nil {
		return err
	}

	// Load diff results
	results, err := store.LoadResults(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load diff results: %w", err)
	}
	if results == nil {
		return fmt.Errorf("session %s has no diff results, please run diff first", sessionID)
	}

	// Generate summary
	summary := diff.FormatDiffSummary(results)
	summary.SessionID = sessionID

	// Create report generator
	rep, err := reporter.NewReporter(reportFormat)
	if err != nil {
		return err
	}

	// Determine output target
	w := os.Stdout
	if reportOutput != "" {
		f, err := os.Create(reportOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	if err := rep.Generate(results, summary, w); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if reportOutput != "" {
		fmt.Printf("Report generated: %s\n", reportOutput)
	}

	return nil
}
