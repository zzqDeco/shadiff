package replay

import (
	"fmt"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/storage"
)

// Engine is the replay engine that coordinates reading and replaying recorded data
type Engine struct {
	store     *storage.FileStore
	sessionID string
	pool      *WorkerPool
	delay     time.Duration
}

// EngineConfig holds the replay engine configuration
type EngineConfig struct {
	SessionID   string
	TargetURL   string
	Concurrency int
	Timeout     time.Duration
	Delay       time.Duration
}

// NewEngine creates a new replay engine
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

// Run executes the replay and returns all replay results
func (e *Engine) Run() ([]ReplayResult, error) {
	// Read recorded records
	records, err := e.store.ListRecords(e.sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read recorded records: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("session %s has no recorded records", e.sessionID)
	}

	logger.ReplayEvent("replay_started",
		"session", e.sessionID,
		"record_count", len(records),
		"concurrency", e.pool.concurrency,
	)

	fmt.Printf("Starting replay: %d records, concurrency: %d\n", len(records), e.pool.concurrency)

	// Execute replay
	results := e.pool.Execute(records, e.delay)

	// Save replay records
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

	fmt.Printf("Replay completed: %d succeeded, %d failed\n", successCount, errorCount)
	return results, nil
}
