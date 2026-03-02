package cmd

import (
	"fmt"
	"os"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/replay"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var (
	replaySession     string
	replayTarget      string
	replayConcurrency int
	replayDelay       string
	replayDBProxy     []string
)

var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "回放录制的流量",
	Long: `读取录制会话中的请求，依次发送到目标服务，并记录新的响应。

示例:
  shadiff replay -s abc123 -t http://new-api:9090
  shadiff replay -s "用户模块迁移" -t http://localhost:9090 -c 5`,
	RunE: runReplay,
}

func init() {
	replayCmd.Flags().StringVarP(&replaySession, "session", "s", "", "会话 ID 或名称 (必填)")
	replayCmd.Flags().StringVarP(&replayTarget, "target", "t", "", "回放目标地址 (必填, e.g. http://localhost:9090)")
	replayCmd.Flags().IntVarP(&replayConcurrency, "concurrency", "c", 1, "并发数")
	replayCmd.Flags().StringVar(&replayDelay, "delay", "", "请求间延迟 (e.g. 100ms)")
	replayCmd.Flags().StringArrayVar(&replayDBProxy, "db-proxy", nil, "DB 代理 (e.g. mysql://:13307->:3306)")

	replayCmd.MarkFlagRequired("session")
	replayCmd.MarkFlagRequired("target")
	rootCmd.AddCommand(replayCmd)
}

func runReplay(cmd *cobra.Command, args []string) error {
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
	sessionID, err := resolveSession(store, replaySession)
	if err != nil {
		return err
	}

	// 解析延迟
	var delay time.Duration
	if replayDelay != "" {
		delay, err = time.ParseDuration(replayDelay)
		if err != nil {
			return fmt.Errorf("无效的延迟: %w", err)
		}
	}

	// 创建回放引擎
	engine := replay.NewEngine(store, replay.EngineConfig{
		SessionID:   sessionID,
		TargetURL:   replayTarget,
		Concurrency: replayConcurrency,
		Delay:       delay,
	})

	// 执行回放
	results, err := engine.Run()
	if err != nil {
		return fmt.Errorf("回放失败: %w", err)
	}

	// 更新会话状态
	session, err := store.Get(sessionID)
	if err == nil {
		session.Status = model.SessionReplayed
		session.Target = model.EndpointConfig{BaseURL: replayTarget}
		store.Update(session)
	}

	// 打印摘要
	errorCount := 0
	for _, r := range results {
		if r.Error != nil {
			errorCount++
		}
	}
	fmt.Printf("\n回放摘要: 总计 %d, 成功 %d, 失败 %d\n", len(results), len(results)-errorCount, errorCount)

	return nil
}

// resolveSession 通过 ID 或名称查找会话
func resolveSession(store *storage.FileStore, nameOrID string) (string, error) {
	// 先尝试直接用 ID
	sess, err := store.Get(nameOrID)
	if err == nil {
		return sess.ID, nil
	}

	// 按名称搜索
	sessions, err := store.List(&model.SessionFilter{Name: nameOrID})
	if err != nil {
		return "", fmt.Errorf("查询会话失败: %w", err)
	}
	if len(sessions) == 0 {
		return "", fmt.Errorf("未找到会话: %s", nameOrID)
	}
	if len(sessions) > 1 {
		fmt.Printf("找到多个匹配会话，使用最新的: %s (%s)\n", sessions[0].ID, sessions[0].Name)
	}
	return sessions[0].ID, nil
}
