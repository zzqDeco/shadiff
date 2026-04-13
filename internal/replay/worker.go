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
	"shadiff/internal/storage"

	"github.com/google/uuid"
)

// WorkerPool is a concurrent replay worker pool
type WorkerPool struct {
	concurrency int
	retryCount  int
	client      *http.Client
	transform   TransformConfig
	store       *storage.FileStore
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(store *storage.FileStore, concurrency int, timeout time.Duration, retryCount int, transform TransformConfig) *WorkerPool {
	return &WorkerPool{
		concurrency: concurrency,
		retryCount:  retryCount,
		client: &http.Client{
			Timeout: timeout,
		},
		transform: transform,
		store:     store,
	}
}

// ReplayResult holds the result of a single replay
type ReplayResult struct {
	Original   model.Record // original recorded record
	Replayed   model.Record // record obtained from replay
	Error      error        // replay error
	StartedAt  int64
	FinishedAt int64
}

// Execute replays a batch of records concurrently
func (wp *WorkerPool) Execute(records []model.Record, delay time.Duration) []ReplayResult {
	results := make([]ReplayResult, len(records))

	if wp.concurrency <= 1 {
		// Sequential replay
		for i, rec := range records {
			results[i] = wp.replayOne(rec)
			if delay > 0 && i < len(records)-1 {
				time.Sleep(delay)
			}
		}
		return results
	}

	// Concurrent replay
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

// replayOne replays a single record
func (wp *WorkerPool) replayOne(original model.Record) ReplayResult {
	result := ReplayResult{Original: original}

	startTime := time.Now()
	result.StartedAt = startTime.UnixMilli()
	httpReq, err := wp.buildReplayRequest(original)
	if httpReq == nil {
		if err == nil {
			err = fmt.Errorf("failed to build request for record %s", original.ID)
		}
		result.Error = err
		result.FinishedAt = time.Now().UnixMilli()
		result.Replayed = model.Record{
			ID:          uuid.New().String()[:8],
			Sequence:    original.Sequence,
			Request:     original.Request,
			SideEffects: []model.SideEffect{},
			RecordedAt:  result.FinishedAt,
			Error:       err.Error(),
		}
		return result
	}

	resp, err := wp.client.Do(httpReq)
	if err != nil && httpReq.Body != nil {
		_ = httpReq.Body.Close()
	}
	for attempt := 0; err != nil && attempt < wp.retryCount; attempt++ {
		httpReq, buildErr := wp.buildReplayRequest(original)
		if buildErr != nil {
			err = buildErr
			break
		}
		resp, err = wp.client.Do(httpReq)
		if err != nil && httpReq.Body != nil {
			_ = httpReq.Body.Close()
		}
	}
	duration := time.Since(startTime).Milliseconds()
	result.FinishedAt = time.Now().UnixMilli()

	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		result.Replayed = model.Record{
			ID:          uuid.New().String()[:8],
			Sequence:    original.Sequence,
			Request:     original.Request,
			SideEffects: []model.SideEffect{},
			Duration:    duration,
			RecordedAt:  result.FinishedAt,
			Error:       err.Error(),
		}
		return result
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("read response body: %w", err)
		result.FinishedAt = time.Now().UnixMilli()
		result.Replayed = model.Record{
			ID:       uuid.New().String()[:8],
			Sequence: original.Sequence,
			Request:  original.Request,
			Response: model.HTTPResponse{
				StatusCode: resp.StatusCode,
				Headers:    cloneHTTPHeaders(resp.Header),
			},
			SideEffects: []model.SideEffect{},
			Duration:    duration,
			RecordedAt:  result.FinishedAt,
			Error:       result.Error.Error(),
		}
		return result
	}
	result.FinishedAt = time.Now().UnixMilli()

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
		RecordedAt:  result.FinishedAt,
	}

	logger.ReplayEvent("request_replayed",
		"sequence", original.Sequence,
		"method", original.Request.Method,
		"path", original.Request.Path,
		"status", resp.StatusCode,
		"duration_ms", duration,
	)

	return result
}

func (wp *WorkerPool) buildReplayRequest(original model.Record) (*http.Request, error) {
	if original.Request.BodyRef != "" {
		if wp.store == nil {
			return nil, fmt.Errorf("request body artifact requires replay storage")
		}

		body, err := wp.store.OpenRequestBodyArtifact(original.SessionID, original.Request.BodyRef)
		if err != nil {
			return nil, fmt.Errorf("open request body artifact: %w", err)
		}

		httpReq := TransformWithBody(original.Request, wp.transform, body, original.Request.BodyLen)
		if httpReq == nil {
			_ = body.Close()
			return nil, fmt.Errorf("failed to build request for record %s", original.ID)
		}
		return httpReq, nil
	}

	var body io.ReadCloser = http.NoBody
	contentLength := int64(0)
	if len(original.Request.Body) > 0 {
		body = io.NopCloser(bytes.NewReader(original.Request.Body))
		contentLength = int64(len(original.Request.Body))
	}

	httpReq := TransformWithBody(original.Request, wp.transform, body, contentLength)
	if httpReq == nil {
		if body != http.NoBody {
			_ = body.Close()
		}
		return nil, fmt.Errorf("failed to build request for record %s", original.ID)
	}
	return httpReq, nil
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
