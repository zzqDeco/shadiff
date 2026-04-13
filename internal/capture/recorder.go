package capture

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"
)

// Recorder is a unified recorder that receives Records and persists them to storage
type Recorder struct {
	sessionID string
	store     *storage.FileStore
	count     atomic.Int64
	scopeID   atomic.Int64

	// sideEffectCh receives side-effect events from DB hooks, etc.
	sideEffectCh chan model.SideEffect
	flushCh      chan chan struct{}
	mu           sync.Mutex
	activeScopes map[int64]*requestScope

	done          chan struct{}
	stopOnce      sync.Once
	collectorDone sync.WaitGroup
}

type requestScope struct {
	startedAt int64
	closedAt  int64
	effects   []model.SideEffect
}

// NewRecorder creates a new recorder
func NewRecorder(sessionID string, store *storage.FileStore) *Recorder {
	r := &Recorder{
		sessionID:    sessionID,
		store:        store,
		sideEffectCh: make(chan model.SideEffect, 1000),
		flushCh:      make(chan chan struct{}),
		activeScopes: make(map[int64]*requestScope),
		done:         make(chan struct{}),
	}
	r.collectorDone.Add(1)
	go r.collectSideEffects()
	return r
}

// BeginRequestScope opens a request attribution scope and returns its ID.
func (r *Recorder) BeginRequestScope(startedAt int64) int64 {
	scopeID := r.scopeID.Add(1)

	r.mu.Lock()
	r.activeScopes[scopeID] = &requestScope{startedAt: startedAt}
	r.mu.Unlock()

	return scopeID
}

// FinishRequestScope closes a request scope, attaches attributed side effects,
// and persists the resulting record.
func (r *Recorder) FinishRequestScope(scopeID int64, record *model.Record) error {
	if record.RecordedAt == 0 {
		record.RecordedAt = time.Now().UnixMilli()
	}

	r.mu.Lock()
	scope, ok := r.activeScopes[scopeID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("finish request scope %d: not found", scopeID)
	}
	scope.closedAt = record.RecordedAt
	r.mu.Unlock()

	r.flushSideEffects()

	r.mu.Lock()
	scope, ok = r.activeScopes[scopeID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("finish request scope %d: not found", scopeID)
	}
	if len(scope.effects) > 0 {
		record.SideEffects = append(record.SideEffects, scope.effects...)
	}
	delete(r.activeScopes, scopeID)
	r.mu.Unlock()

	return r.persistRecord(record)
}

// Record records a single behavior entry without request-scope attribution.
func (r *Recorder) Record(record *model.Record) error {
	return r.persistRecord(record)
}

func (r *Recorder) persistRecord(record *model.Record) error {
	record.SessionID = r.sessionID

	if err := r.store.AppendRecord(r.sessionID, record); err != nil {
		return fmt.Errorf("append record: %w", err)
	}

	count := r.count.Add(1)
	logger.CaptureEvent("record_saved",
		"session", r.sessionID,
		"record_id", record.ID,
		"count", count,
	)
	return nil
}

// SideEffectChan returns the side-effect channel for external components like DB hooks to send side effects
func (r *Recorder) SideEffectChan() chan<- model.SideEffect {
	return r.sideEffectCh
}

// Count returns the number of recorded entries
func (r *Recorder) Count() int64 {
	return r.count.Load()
}

// Stop stops the recorder
func (r *Recorder) Stop() {
	r.stopOnce.Do(func() {
		close(r.done)
		r.collectorDone.Wait()
	})
}

// collectSideEffects collects side-effect events in the background
func (r *Recorder) collectSideEffects() {
	defer r.collectorDone.Done()

	for {
		select {
		case effect := <-r.sideEffectCh:
			r.assignSideEffect(effect)
		case flushDone := <-r.flushCh:
			r.drainSideEffects()
			close(flushDone)
		case <-r.done:
			r.drainSideEffects()
			return
		}
	}
}

func (r *Recorder) flushSideEffects() {
	flushDone := make(chan struct{})

	select {
	case r.flushCh <- flushDone:
		<-flushDone
	case <-r.done:
	}
}

func (r *Recorder) drainSideEffects() {
	for {
		select {
		case effect := <-r.sideEffectCh:
			r.assignSideEffect(effect)
		default:
			return
		}
	}
}

func (r *Recorder) assignSideEffect(effect model.SideEffect) {
	r.mu.Lock()
	defer r.mu.Unlock()

	scopeID, scope := r.findScopeForEffect(effect.Timestamp)
	if scope == nil {
		logger.Warn("orphan side effect dropped",
			"session", r.sessionID,
			"db_type", effect.DBType,
			"timestamp", effect.Timestamp,
			"query", effect.Query,
		)
		return
	}

	scope.effects = append(scope.effects, effect)
	logger.Debug("side effect attributed",
		"session", r.sessionID,
		"scope_id", scopeID,
		"timestamp", effect.Timestamp,
	)
}

func (r *Recorder) findScopeForEffect(timestamp int64) (int64, *requestScope) {
	var (
		matchID    int64
		matchScope *requestScope
	)

	for scopeID, scope := range r.activeScopes {
		if scope.startedAt > timestamp {
			continue
		}
		if scope.closedAt != 0 && timestamp > scope.closedAt {
			continue
		}

		if matchScope == nil || scope.startedAt > matchScope.startedAt || (scope.startedAt == matchScope.startedAt && scopeID > matchID) {
			matchID = scopeID
			matchScope = scope
		}
	}

	return matchID, matchScope
}
