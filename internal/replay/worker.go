package replay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"

	"github.com/google/uuid"
)

// WorkerPool 并发回放工作池
type WorkerPool struct {
	concurrency int
	client      *http.Client
	transform   TransformConfig
}

// NewWorkerPool 创建工作池
func NewWorkerPool(concurrency int, timeout time.Duration, transform TransformConfig) *WorkerPool {
	return &WorkerPool{
		concurrency: concurrency,
		client: &http.Client{
			Timeout: timeout,
		},
		transform: transform,
	}
}

// ReplayResult 单次回放结果
type ReplayResult struct {
	Original model.Record // 原始录制记录
	Replayed model.Record // 回放得到的记录
	Error    error        // 回放错误
}

// Execute 并发回放一批记录
func (wp *WorkerPool) Execute(records []model.Record, delay time.Duration) []ReplayResult {
	results := make([]ReplayResult, len(records))

	if wp.concurrency <= 1 {
		// 串行回放
		for i, rec := range records {
			results[i] = wp.replayOne(rec)
			if delay > 0 && i < len(records)-1 {
				time.Sleep(delay)
			}
		}
		return results
	}

	// 并发回放
	jobs := make(chan int, len(records))
	var wg sync.WaitGroup

	for w := 0; w < wp.concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx] = wp.replayOne(records[idx])
				if delay > 0 {
					time.Sleep(delay)
				}
			}
		}()
	}

	for i := range records {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return results
}

// replayOne 回放单条记录
func (wp *WorkerPool) replayOne(original model.Record) ReplayResult {
	result := ReplayResult{Original: original}

	httpReq := Transform(original.Request, wp.transform)
	if httpReq == nil {
		result.Error = fmt.Errorf("failed to build request for record %s", original.ID)
		return result
	}

	startTime := time.Now()
	resp, err := wp.client.Do(httpReq)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		result.Replayed = model.Record{
			ID:         uuid.New().String()[:8],
			Sequence:   original.Sequence,
			Request:    original.Request,
			Duration:   duration,
			RecordedAt: time.Now().UnixMilli(),
			Error:      err.Error(),
		}
		return result
	}
	defer resp.Body.Close()

	// 读取响应 body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("read response body: %w", err)
		return result
	}

	result.Replayed = model.Record{
		ID:       uuid.New().String()[:8],
		Sequence: original.Sequence,
		Request:  original.Request,
		Response: model.HTTPResponse{
			StatusCode: resp.StatusCode,
			Headers:    cloneHTTPHeaders(resp.Header),
			Body:       respBody,
			BodyLen:    int64(len(respBody)),
		},
		SideEffects: []model.SideEffect{},
		Duration:    duration,
		RecordedAt:  time.Now().UnixMilli(),
	}

	logger.ReplayEvent("request_replayed",
		"sequence", original.Sequence,
		"method", original.Request.Method,
		"path", original.Request.Path,
		"status", resp.StatusCode,
		"duration_ms", duration,
	)

	// 将请求 body 复位以供后续使用
	_ = bytes.NewReader(original.Request.Body)

	return result
}

func cloneHTTPHeaders(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}
	result := make(map[string][]string, len(h))
	for k, v := range h {
		cp := make([]string, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}
