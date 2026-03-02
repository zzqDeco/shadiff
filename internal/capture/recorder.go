package capture

import (
	"fmt"
	"sync"
	"sync/atomic"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"
)

// Recorder 统一录制器，接收 Record 并持久化到存储
type Recorder struct {
	sessionID string
	store     *storage.FileStore
	count     atomic.Int64

	// sideEffectCh 接收来自 DB hook 等的副作用事件
	sideEffectCh chan model.SideEffect
	// pendingEffects 暂存尚未关联到 Record 的副作用
	pendingEffects []model.SideEffect
	mu             sync.Mutex

	done chan struct{}
}

// NewRecorder 创建录制器
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

// Record 录制一条行为记录
func (r *Recorder) Record(record *model.Record) error {
	record.SessionID = r.sessionID

	// 将暂存的副作用附加到 record
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

// SideEffectChan 返回副作用通道，供 DB hook 等外部组件发送副作用
func (r *Recorder) SideEffectChan() chan<- model.SideEffect {
	return r.sideEffectCh
}

// Count 返回已录制的记录数
func (r *Recorder) Count() int64 {
	return r.count.Load()
}

// Stop 停止录制器
func (r *Recorder) Stop() {
	close(r.done)
}

// collectSideEffects 后台收集副作用事件
func (r *Recorder) collectSideEffects() {
	for {
		select {
		case effect := <-r.sideEffectCh:
			r.mu.Lock()
			r.pendingEffects = append(r.pendingEffects, effect)
			r.mu.Unlock()
		case <-r.done:
			// 排空通道
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
