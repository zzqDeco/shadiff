package replay

import (
	"context"
	"fmt"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"
)

// Engine is the replay engine that coordinates reading and replaying recorded data
type Engine struct {
	store              *storage.FileStore
	sessionID          string
	pool               *WorkerPool
	delay              time.Duration
	sideEffectCh       <-chan model.SideEffect
	pendingSideEffects []model.SideEffect
	flusher            sideEffectFlusher
	flushTimeout       time.Duration
}

type sideEffectFlusher interface {
	Flush(context.Context) error
}

// EngineConfig holds the replay engine configuration
type EngineConfig struct {
	SessionID    string
	TargetURL    string
	Concurrency  int
	Timeout      time.Duration
	RetryCount   int
	Delay        time.Duration
	SideEffectCh <-chan model.SideEffect
	Flusher      sideEffectFlusher
	FlushTimeout time.Duration
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
		store:        store,
		sessionID:    cfg.SessionID,
		pool:         NewWorkerPool(store, concurrency, timeout, cfg.RetryCount, transform),
		delay:        cfg.Delay,
		sideEffectCh: cfg.SideEffectCh,
		flusher:      cfg.Flusher,
		flushTimeout: cfg.FlushTimeout,
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

	if e.sideEffectCh != nil && e.pool.concurrency > 1 {
		return nil, fmt.Errorf("replay db side-effect capture requires concurrency 1")
	}

	var results []ReplayResult
	if e.sideEffectCh != nil {
		results = e.executeSequentialWithSideEffects(records)
		e.dropPendingSideEffects()
	} else {
		results = e.pool.Execute(records, e.delay)
	}

	// Save replay records
	successCount := 0
	errorCount := 0
	for _, r := range results {
		if r.Error != nil {
			errorCount++
		} else {
			successCount++
		}
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

func (e *Engine) executeSequentialWithSideEffects(records []model.Record) []ReplayResult {
	results := make([]ReplayResult, len(records))

	for i, rec := range records {
		result := e.pool.replayOne(rec)
		e.flushSideEffects()
		result.Replayed.SideEffects = e.takeSideEffectsWindow(result.StartedAt, result.FinishedAt)
		results[i] = result
		if e.delay > 0 && i < len(records)-1 {
			time.Sleep(e.delay)
		}
	}

	return results
}

func (e *Engine) takeSideEffectsWindow(startedAt, finishedAt int64) []model.SideEffect {
	e.drainSideEffects()

	var (
		matched []model.SideEffect
		kept    []model.SideEffect
	)

	for _, effect := range e.pendingSideEffects {
		switch {
		case effect.Timestamp < startedAt:
			logger.Warn("replay orphan side effect dropped",
				"session", e.sessionID,
				"db_type", effect.DBType,
				"timestamp", effect.Timestamp,
				"query", effect.Query,
			)
		case effect.Timestamp <= finishedAt:
			matched = append(matched, effect)
		default:
			kept = append(kept, effect)
		}
	}

	e.pendingSideEffects = kept
	return matched
}

func (e *Engine) dropPendingSideEffects() {
	e.drainSideEffects()
	for _, effect := range e.pendingSideEffects {
		logger.Warn("replay orphan side effect dropped",
			"session", e.sessionID,
			"db_type", effect.DBType,
			"timestamp", effect.Timestamp,
			"query", effect.Query,
		)
	}
	e.pendingSideEffects = nil
}

func (e *Engine) drainSideEffects() {
	for e.sideEffectCh != nil {
		select {
		case effect, ok := <-e.sideEffectCh:
			if !ok {
				e.sideEffectCh = nil
				return
			}
			e.pendingSideEffects = append(e.pendingSideEffects, effect)
		default:
			return
		}
	}
}

func (e *Engine) flushSideEffects() {
	if e.flusher == nil {
		return
	}

	ctx := context.Background()
	cancel := func() {}
	if e.flushTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, e.flushTimeout)
	}
	defer cancel()

	if err := e.flusher.Flush(ctx); err != nil {
		logger.Warn("replay side-effect flush failed", "error", err.Error())
	}
}
