package replay

import (
	"fmt"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/storage"
)

// Engine 回放引擎，协调录制数据的读取和回放
type Engine struct {
	store     *storage.FileStore
	sessionID string
	pool      *WorkerPool
	delay     time.Duration
}

// EngineConfig 回放引擎配置
type EngineConfig struct {
	SessionID   string
	TargetURL   string
	Concurrency int
	Timeout     time.Duration
	Delay       time.Duration
}

// NewEngine 创建回放引擎
func NewEngine(store *storage.FileStore, cfg EngineConfig) *Engine {
	transform := TransformConfig{
		TargetBaseURL: cfg.TargetURL,
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	return &Engine{
		store:     store,
		sessionID: cfg.SessionID,
		pool:      NewWorkerPool(concurrency, timeout, transform),
		delay:     cfg.Delay,
	}
}

// Run 执行回放，返回所有回放结果
func (e *Engine) Run() ([]ReplayResult, error) {
	// 读取录制记录
	records, err := e.store.ListRecords(e.sessionID)
	if err != nil {
		return nil, fmt.Errorf("读取录制记录失败: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("会话 %s 没有录制记录", e.sessionID)
	}

	logger.ReplayEvent("replay_started",
		"session", e.sessionID,
		"record_count", len(records),
		"concurrency", e.pool.concurrency,
	)

	fmt.Printf("开始回放: %d 条记录, 并发数: %d\n", len(records), e.pool.concurrency)

	// 执行回放
	results := e.pool.Execute(records, e.delay)

	// 保存回放记录
	successCount := 0
	errorCount := 0
	for _, r := range results {
		if r.Error != nil {
			errorCount++
			continue
		}
		successCount++
		if err := e.store.AppendReplayRecord(e.sessionID, &r.Replayed); err != nil {
			logger.Error("save replay record failed", err, "sequence", r.Original.Sequence)
		}
	}

	logger.ReplayEvent("replay_completed",
		"session", e.sessionID,
		"total", len(results),
		"success", successCount,
		"errors", errorCount,
	)

	fmt.Printf("回放完成: 成功 %d, 失败 %d\n", successCount, errorCount)
	return results, nil
}
