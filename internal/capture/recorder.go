package capture

import (
	"fmt"
	"sync"
	"sync/atomic"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"
)

// Recorder is a unified recorder that receives Records and persists them to storage
type Recorder struct {
	sessionID string
	store     *storage.FileStore
	count     atomic.Int64

	// sideEffectCh receives side-effect events from DB hooks, etc.
	sideEffectCh chan model.SideEffect
	// pendingEffects temporarily stores side effects not yet associated with a Record
	pendingEffects []model.SideEffect
	mu             sync.Mutex

	done chan struct{}
}

// NewRecorder creates a new recorder
func NewRecorder(sessionID string, store *storage.FileStore) *Recorder {
	r := &Recorder{
		sessionID:    sessionID,
		store:        store,
		sideEffectCh: make(chan model.SideEffect, 1000),
		done:         make(chan struct{}),
	}
	go r.collectSideEffects()
	return r
}

// Record records a single behavior entry
func (r *Recorder) Record(record *model.Record) error {
	record.SessionID = r.sessionID

	// Attach pending side effects to the record
	r.mu.Lock()
	if len(r.pendingEffects) > 0 {
		record.SideEffects = append(record.SideEffects, r.pendingEffects...)
		r.pendingEffects = nil
	}
	r.mu.Unlock()

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
	close(r.done)
}

// collectSideEffects collects side-effect events in the background
func (r *Recorder) collectSideEffects() {
	for {
		select {
		case effect := <-r.sideEffectCh:
			r.mu.Lock()
			r.pendingEffects = append(r.pendingEffects, effect)
			r.mu.Unlock()
		case <-r.done:
			// Drain the channel
			for {
				select {
				case effect := <-r.sideEffectCh:
					r.mu.Lock()
					r.pendingEffects = append(r.pendingEffects, effect)
					r.mu.Unlock()
				default:
					return
				}
			}
		}
	}
}
