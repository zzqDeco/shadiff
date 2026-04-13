package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"shadiff/internal/diff"
	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/reporter"
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
	cfg := currentConfig()
	diffIgnoreOrder = effectiveBool(cmd.Flags().Changed("ignore-order"), diffIgnoreOrder, cfg.Diff.IgnoreOrder)
	diffIgnoreHeaders = effectiveStrings(cmd.Flags().Changed("ignore-headers"), diffIgnoreHeaders, cfg.Diff.IgnoreHeaders)
	diffRulesFile = effectiveString(cmd.Flags().Changed("rules"), diffRulesFile, cfg.Diff.RulesFile)

	// Initialize logger
	if err := logger.Init(currentLogDir(), effectiveLogLevel()); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()

	// Create storage
	store, err := storage.NewFileStore(currentDataDir())
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Find session
	sessionID, err := resolveSession(store, diffSession)
	if err != nil {
		return err
	}

	rules := diff.RulesFromConfig(cfg.Diff.Rules)
	if diffRulesFile != "" {
		fileRules, err := diff.LoadRulesFile(diffRulesFile)
		if err != nil {
			return fmt.Errorf("load rules file: %w", err)
		}
		rules = append(rules, fileRules...)
	}

	// Create diff engine
	engine := diff.NewEngine(store, diff.EngineConfig{
		SessionID:     sessionID,
		Rules:         rules,
		IgnoreOrder:   diffIgnoreOrder,
		IgnoreHeaders: diffIgnoreHeaders,
		MaxDiffs:      cfg.Diff.MaxDiffs,
	})

	// Execute diff
	results, err := engine.Run()
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	summary := diff.FormatDiffSummary(results)
	summary.SessionID = sessionID

	if err := writeDiffResults(cmd, results, summary, os.Stdout); err != nil {
		return err
	}

	return nil
}

func writeDiffResults(cmd *cobra.Command, results []model.DiffResult, summary model.DiffSummary, w io.Writer) error {
	switch strings.ToLower(currentDiffOutput(cmd)) {
	case "", "terminal":
		printDiffResults(w, results, summary)
		return nil
	case "json":
		return (&reporter.JSONReporter{}).Generate(results, summary, w)
	default:
		return fmt.Errorf("unsupported diff output format: %s", currentDiffOutput(cmd))
	}
}

func currentDiffOutput(cmd *cobra.Command) string {
	if cmd != nil {
		flags := cmd.Flags()
		if flags != nil && flags.Lookup("output") != nil {
			if output, err := flags.GetString("output"); err == nil && output != "" {
				return output
			}
		}
	}
	if diffOutput == "" {
		return "terminal"
	}
	return diffOutput
}

func printDiffResults(w io.Writer, results []model.DiffResult, summary model.DiffSummary) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "━━ Diff Report ━━")

	for _, r := range results {
		method := r.Request.Method
		path := r.Request.Path

		if r.Match {
			fmt.Fprintf(w, "  ✔ %-7s %s  [MATCH]\n", method, path)
		} else {
			fmt.Fprintf(w, "  ✘ %-7s %s  [DIFF]\n", method, path)
			for _, d := range r.Differences {
				if d.Ignored {
					fmt.Fprintf(w, "    ├ %s: ignored(%s)\n", d.Path, d.Rule)
				} else {
					severity := "error"
					switch d.Severity {
					case model.SeverityWarning:
						severity = "warning"
					case model.SeverityInfo:
						severity = "info"
					}
					if d.Path != "" {
						fmt.Fprintf(w, "    └ %s: %v ≠ %v\n", d.Path, d.Expected, d.Actual)
					} else {
						fmt.Fprintf(w, "    └ %s\n", d.Message)
					}
					fmt.Fprintf(w, "      severity: %s\n", severity)
				}
			}
		}
	}

	// Summary
	fmt.Fprintln(w, "────────────────")
	fmt.Fprintf(w, "Total: %d records, %d matched, %d differences\n",
		summary.TotalCount, summary.MatchCount, summary.DiffCount)
	fmt.Fprintf(w, "Match rate: %.0f%%\n", summary.MatchRate*100)
}
