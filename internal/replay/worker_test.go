package replay

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"shadiff/internal/model"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func makeReplayRecord() model.Record {
	return model.Record{
		ID:       "rec1",
		Sequence: 1,
		Request: model.HTTPRequest{
			Method: "GET",
			Path:   "/v1/test",
		},
	}
}

func TestWorkerPool_RetriesNetworkErrors(t *testing.T) {
	attempts := 0
	wp := NewWorkerPool(1, 0, 2, TransformConfig{TargetBaseURL: "http://example.com"})
	wp.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return nil, errors.New("temporary network error")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
	}

	result := wp.replayOne(makeReplayRecord())
	if result.Error != nil {
		t.Fatalf("replayOne() error = %v", result.Error)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want %d", attempts, 3)
	}
}

func TestWorkerPool_DoesNotRetryHTTPStatus(t *testing.T) {
	attempts := 0
	wp := NewWorkerPool(1, 0, 3, TransformConfig{TargetBaseURL: "http://example.com"})
	wp.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		}),
	}

	result := wp.replayOne(makeReplayRecord())
	if result.Error != nil {
		t.Fatalf("replayOne() error = %v", result.Error)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want %d", attempts, 1)
	}
	if result.Replayed.Response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", result.Replayed.Response.StatusCode, http.StatusInternalServerError)
	}
}
