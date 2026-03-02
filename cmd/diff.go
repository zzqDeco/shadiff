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
	Short: "对比录制和回放的行为差异",
	Long: `读取录制和回放记录，进行语义级对比，输出差异报告。

示例:
  shadiff diff -s abc123
  shadiff diff -s "用户模块迁移" --ignore-order -r rules.yaml`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVarP(&diffSession, "session", "s", "", "会话 ID 或名称 (必填)")
	diffCmd.Flags().StringVarP(&diffRulesFile, "rules", "r", "", "对拍规则文件 (JSON/YAML)")
	diffCmd.Flags().BoolVar(&diffIgnoreOrder, "ignore-order", false, "忽略 JSON 数组顺序")
	diffCmd.Flags().StringArrayVar(&diffIgnoreHeaders, "ignore-headers", nil, "额外忽略的 header")
	diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "terminal", "输出格式: terminal, json")

	diffCmd.MarkFlagRequired("session")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	// 初始化日志
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/.shadiff"
	if err := logger.Init(dataDir); err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}
	defer logger.Close()

	// 创建存储
	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		return fmt.Errorf("创建存储失败: %w", err)
	}

	// 查找会话
	sessionID, err := resolveSession(store, diffSession)
	if err != nil {
		return err
	}

	// 创建对拍引擎
	engine := diff.NewEngine(store, diff.EngineConfig{
		SessionID:     sessionID,
		IgnoreOrder:   diffIgnoreOrder,
		IgnoreHeaders: diffIgnoreHeaders,
	})

	// 执行对拍
	results, err := engine.Run()
	if err != nil {
		return fmt.Errorf("对拍失败: %w", err)
	}

	// 输出结果
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
					fmt.Printf("    ├ %s: 忽略(%s)\n", d.Path, d.Rule)
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

	// 摘要
	summary := diff.FormatDiffSummary(results)
	fmt.Println("────────────────")
	fmt.Printf("总计: %d 条记录, %d 匹配, %d 差异\n",
		summary.TotalCount, summary.MatchCount, summary.DiffCount)
	fmt.Printf("匹配率: %.0f%%\n", summary.MatchRate*100)
}
