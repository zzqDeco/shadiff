package replay

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"shadiff/internal/model"
	"shadiff/internal/storage"
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
	wp := NewWorkerPool(nil, 1, 0, 2, TransformConfig{TargetBaseURL: "http://example.com"})
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
	wp := NewWorkerPool(nil, 1, 0, 3, TransformConfig{TargetBaseURL: "http://example.com"})
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

func TestWorkerPool_ReplaysFromRequestBodyArtifact(t *testing.T) {
	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	session := &model.Session{Name: "artifact", Status: model.SessionRecording}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	fullBody := []byte("full request body from artifact")
	ref, err := store.SaveRequestBodyArtifact(session.ID, "rec1", bytes.NewReader(fullBody))
	if err != nil {
		t.Fatalf("SaveRequestBodyArtifact() error: %v", err)
	}

	record := makeReplayRecord()
	record.SessionID = session.ID
	record.Request.Method = http.MethodPost
	record.Request.Body = []byte("full ")
	record.Request.BodyLen = int64(len(fullBody))
	record.Request.BodyRef = ref

	wp := NewWorkerPool(store, 1, 0, 0, TransformConfig{TargetBaseURL: "http://example.com"})
	wp.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(req.Body) error: %v", err)
			}
			if !bytes.Equal(body, fullBody) {
				t.Fatalf("request body = %q, want %q", string(body), string(fullBody))
			}
			if req.ContentLength != int64(len(fullBody)) {
				t.Fatalf("ContentLength = %d, want %d", req.ContentLength, len(fullBody))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ok")),
			}, nil
		}),
	}

	result := wp.replayOne(record)
	if result.Error != nil {
		t.Fatalf("replayOne() error = %v", result.Error)
	}
}
