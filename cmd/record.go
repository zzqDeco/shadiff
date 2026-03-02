package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shadiff/internal/capture"
	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"

	"github.com/spf13/cobra"
)

var (
	recordTarget  string
	recordListen  string
	recordSession string
	recordDBProxy []string
	recordDuration string
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "启动流量录制",
	Long: `启动 HTTP 反向代理，录制经过代理的所有请求/响应及数据库副作用。

示例:
  shadiff record -t http://localhost:8080 -l :18080 -s "用户模块迁移"
  shadiff record -t http://old-api:8080 --db-proxy mysql://:13306->:3306`,
	RunE: runRecord,
}

func init() {
	recordCmd.Flags().StringVarP(&recordTarget, "target", "t", "", "目标服务地址 (必填, e.g. http://localhost:8080)")
	recordCmd.Flags().StringVarP(&recordListen, "listen", "l", ":18080", "代理监听地址")
	recordCmd.Flags().StringVarP(&recordSession, "session", "s", "", "会话名称 (默认自动生成)")
	recordCmd.Flags().StringArrayVar(&recordDBProxy, "db-proxy", nil, "DB 代理 (e.g. mysql://:13306->:3306)")
	recordCmd.Flags().StringVarP(&recordDuration, "duration", "d", "", "最大录制时长 (e.g. 30m)")

	recordCmd.MarkFlagRequired("target")
	rootCmd.AddCommand(recordCmd)
}

func runRecord(cmd *cobra.Command, args []string) error {
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

	// 创建会话
	sessionName := recordSession
	if sessionName == "" {
		sessionName = fmt.Sprintf("record-%s", time.Now().Format("20060102-150405"))
	}

	session := &model.Session{
		Name:   sessionName,
		Status: model.SessionRecording,
		Source: model.EndpointConfig{
			BaseURL: recordTarget,
		},
	}

	if err := store.Create(session); err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}

	fmt.Printf("会话已创建: %s (%s)\n", session.ID, session.Name)

	// 创建录制器
	recorder := capture.NewRecorder(session.ID, store)
	defer recorder.Stop()

	// 创建代理
	proxy, err := capture.NewProxy(recordTarget, recorder)
	if err != nil {
		return fmt.Errorf("创建代理失败: %w", err)
	}

	// 启动 HTTP 服务
	server := &http.Server{
		Addr:    recordListen,
		Handler: proxy,
	}

	// 上下文和信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理超时
	if recordDuration != "" {
		dur, err := time.ParseDuration(recordDuration)
		if err != nil {
			return fmt.Errorf("无效的时长: %w", err)
		}
		ctx, cancel = context.WithTimeout(ctx, dur)
		defer cancel()
	}

	// 信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务
	go func() {
		fmt.Printf("录制代理已启动: %s -> %s\n", recordListen, recordTarget)
		fmt.Println("按 Ctrl+C 停止录制...")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("server error", err)
		}
	}()

	// 等待停止信号
	select {
	case <-sigCh:
		fmt.Println("\n正在停止录制...")
	case <-ctx.Done():
		fmt.Println("\n录制时间到，正在停止...")
	}

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)

	// 更新会话状态
	recordCount := int(recorder.Count())
	session.Status = model.SessionCompleted
	session.RecordCount = recordCount
	if err := store.Update(session); err != nil {
		logger.Error("update session failed", err)
	}

	fmt.Printf("录制完成: %d 条记录已保存到会话 %s\n", recordCount, session.ID)
	return nil
}
