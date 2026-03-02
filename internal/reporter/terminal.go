package reporter

import (
	"fmt"
	"io"

	"shadiff/internal/model"
)

// TerminalReporter 终端彩色输出报告
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
					fmt.Fprintf(w, "    %s \033[90m%s: 忽略(%s)\033[0m\n", prefix, d.Path, d.Rule)
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

	// 摘要
	fmt.Fprintln(w)
	fmt.Fprintln(w, "────────────────")
	fmt.Fprintf(w, "总计: %d 条记录, %d 匹配, %d 差异\n",
		summary.TotalCount, summary.MatchCount, summary.DiffCount)
	if summary.IgnoreCount > 0 {
		fmt.Fprintf(w, "忽略: %d 条差异 (规则匹配)\n", summary.IgnoreCount)
	}
	if summary.ErrorCount > 0 {
		fmt.Fprintf(w, "严重: %d 条错误级差异\n", summary.ErrorCount)
	}
	fmt.Fprintf(w, "匹配率: \033[1m%.1f%%\033[0m\n", summary.MatchRate*100)
	fmt.Fprintln(w)

	return nil
}
