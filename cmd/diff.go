package cmd

import (
	"fmt"
	"os"

	"shadiff/internal/diff"
	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var (
	diffSession       string
	diffRulesFile     string
	diffIgnoreOrder   bool
	diffIgnoreHeaders []string
	diffOutput        string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare behavioral differences between recorded and replayed traffic",
	Long: `Read recorded and replayed data, perform semantic-level comparison, and output a diff report.

Examples:
  shadiff diff -s abc123
  shadiff diff -s "user-module-migration" --ignore-order -r rules.yaml`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVarP(&diffSession, "session", "s", "", "Session ID or name (required)")
	diffCmd.Flags().StringVarP(&diffRulesFile, "rules", "r", "", "Diff rules file (JSON/YAML)")
	diffCmd.Flags().BoolVar(&diffIgnoreOrder, "ignore-order", false, "Ignore JSON array order")
	diffCmd.Flags().StringArrayVar(&diffIgnoreHeaders, "ignore-headers", nil, "Additional headers to ignore")
	diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "terminal", "Output format: terminal, json")

	diffCmd.MarkFlagRequired("session")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Initialize logger
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/.shadiff"
	if err := logger.Init(dataDir); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()

	// Create storage
	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Find session
	sessionID, err := resolveSession(store, diffSession)
	if err != nil {
		return err
	}

	// Create diff engine
	engine := diff.NewEngine(store, diff.EngineConfig{
		SessionID:     sessionID,
		IgnoreOrder:   diffIgnoreOrder,
		IgnoreHeaders: diffIgnoreHeaders,
	})

	// Execute diff
	results, err := engine.Run()
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	// Output results
	printDiffResults(results)

	return nil
}

func printDiffResults(results []model.DiffResult) {
	fmt.Println()
	fmt.Println("━━ Diff Report ━━")

	for _, r := range results {
		method := r.Request.Method
		path := r.Request.Path

		if r.Match {
			fmt.Printf("  ✔ %-7s %s  [MATCH]\n", method, path)
		} else {
			fmt.Printf("  ✘ %-7s %s  [DIFF]\n", method, path)
			for _, d := range r.Differences {
				if d.Ignored {
					fmt.Printf("    ├ %s: ignored(%s)\n", d.Path, d.Rule)
				} else {
					severity := "error"
					switch d.Severity {
					case model.SeverityWarning:
						severity = "warning"
					case model.SeverityInfo:
						severity = "info"
					}
					if d.Path != "" {
						fmt.Printf("    └ %s: %v ≠ %v\n", d.Path, d.Expected, d.Actual)
					} else {
						fmt.Printf("    └ %s\n", d.Message)
					}
					fmt.Printf("      severity: %s\n", severity)
				}
			}
		}
	}

	// Summary
	summary := diff.FormatDiffSummary(results)
	fmt.Println("────────────────")
	fmt.Printf("Total: %d records, %d matched, %d differences\n",
		summary.TotalCount, summary.MatchCount, summary.DiffCount)
	fmt.Printf("Match rate: %.0f%%\n", summary.MatchRate*100)
}
