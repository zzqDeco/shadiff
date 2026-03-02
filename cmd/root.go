package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "shadiff",
	Short: "影子流量语义对拍工具",
	Long: `Shadiff - 影子流量语义对拍工具

通过黑盒录制-回放-对拍三段式流程，验证跨框架/跨语言 API 迁移的行为一致性。

工作流程:
  1. shadiff record  — 录制老 API 的行为 (请求/响应/DB副作用)
  2. shadiff replay  — 将录制的流量回放到新 API
  3. shadiff diff    — 语义级比较两边的行为差异
  4. shadiff report  — 生成详细的对拍报告`,
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认 ~/.shadiff/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "显示详细日志")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "仅显示错误信息")

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	// 设置版本信息
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
