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
	Short: "生成详细报告",
	Long: `根据对拍结果生成详细报告，支持 terminal/JSON/HTML 格式。

示例:
  shadiff report -s abc123
  shadiff report -s abc123 -f html -o report.html
  shadiff report -s abc123 -f json -o result.json`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().StringVarP(&reportSession, "session", "s", "", "会话 ID 或名称 (必填)")
	reportCmd.Flags().StringVarP(&reportFormat, "format", "f", "terminal", "报告格式: terminal, json, html")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "输出路径 (默认 stdout)")

	reportCmd.MarkFlagRequired("session")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	homeDir, _ := os.UserHomeDir()
	dataDir := homeDir + "/.shadiff"
	if err := logger.Init(dataDir); err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}
	defer logger.Close()

	store, err := storage.NewFileStore(dataDir)
	if err != nil {
		return fmt.Errorf("创建存储失败: %w", err)
	}

	sessionID, err := resolveSession(store, reportSession)
	if err != nil {
		return err
	}

	// 加载对拍结果
	results, err := store.LoadResults(sessionID)
	if err != nil {
		return fmt.Errorf("加载对拍结果失败: %w", err)
	}
	if results == nil {
		return fmt.Errorf("会话 %s 没有对拍结果，请先执行 diff", sessionID)
	}

	// 生成摘要
	summary := diff.FormatDiffSummary(results)
	summary.SessionID = sessionID

	// 创建报告生成器
	rep, err := reporter.NewReporter(reportFormat)
	if err != nil {
		return err
	}

	// 确定输出目标
	w := os.Stdout
	if reportOutput != "" {
		f, err := os.Create(reportOutput)
		if err != nil {
			return fmt.Errorf("创建输出文件失败: %w", err)
		}
		defer f.Close()
		w = f
	}

	if err := rep.Generate(results, summary, w); err != nil {
		return fmt.Errorf("生成报告失败: %w", err)
	}

	if reportOutput != "" {
		fmt.Printf("报告已生成: %s\n", reportOutput)
	}

	return nil
}
