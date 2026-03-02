package reporter

import (
	"fmt"
	"io"

	"shadiff/internal/model"
)

// TerminalReporter outputs colored reports to the terminal
type TerminalReporter struct{}

func (r *TerminalReporter) Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error {
	fmt.Fprintln(w)
	fmt.Fprintln(w, "━━ Shadiff Report ━━")
	fmt.Fprintln(w)

	for _, res := range results {
		method := res.Request.Method
		path := res.Request.Path
		if path == "" {
			path = "/"
		}

		if res.Match {
			fmt.Fprintf(w, "  \033[32m✔\033[0m %-7s %s  \033[32m[MATCH]\033[0m\n", method, path)
		} else {
			fmt.Fprintf(w, "  \033[31m✘\033[0m %-7s %s  \033[31m[DIFF]\033[0m\n", method, path)
			for i, d := range res.Differences {
				prefix := "├"
				if i == len(res.Differences)-1 {
					prefix = "└"
				}

				if d.Ignored {
					fmt.Fprintf(w, "    %s \033[90m%s: ignored(%s)\033[0m\n", prefix, d.Path, d.Rule)
				} else {
					sevColor := "\033[31m" // red for error
					switch d.Severity {
					case model.SeverityWarning:
						sevColor = "\033[33m" // yellow
					case model.SeverityInfo:
						sevColor = "\033[36m" // cyan
					}

					if d.Path != "" {
						fmt.Fprintf(w, "    %s %s: %v ≠ %v\n", prefix, d.Path, d.Expected, d.Actual)
					} else {
						fmt.Fprintf(w, "    %s %s\n", prefix, d.Message)
					}
					fmt.Fprintf(w, "      %sseverity: %s\033[0m\n", sevColor, d.Severity)
				}
			}
		}
	}

	// Summary
	fmt.Fprintln(w)
	fmt.Fprintln(w, "────────────────")
	fmt.Fprintf(w, "Total: %d records, %d matched, %d differences\n",
		summary.TotalCount, summary.MatchCount, summary.DiffCount)
	if summary.IgnoreCount > 0 {
		fmt.Fprintf(w, "Ignored: %d differences (rule matched)\n", summary.IgnoreCount)
	}
	if summary.ErrorCount > 0 {
		fmt.Fprintf(w, "Critical: %d error-level differences\n", summary.ErrorCount)
	}
	fmt.Fprintf(w, "Match rate: \033[1m%.1f%%\033[0m\n", summary.MatchRate*100)
	fmt.Fprintln(w)

	return nil
}
