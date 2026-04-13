package capture

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shadiff/internal/model"
	"shadiff/internal/storage"
)

func newProxyTestStore(t *testing.T) (*storage.FileStore, *model.Session) {
	t.Helper()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	session := &model.Session{
		Name:   "proxy-test",
		Status: model.SessionRecording,
		Source: model.EndpointConfig{BaseURL: "http://source"},
	}
	if err := store.Create(session); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	return store, session
}

func TestProxy_TruncatesBodies(t *testing.T) {
	var upstreamBody string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		upstreamBody = string(body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("resp:" + string(body)))
	}))
	defer target.Close()

	store, session := newProxyTestStore(t)
	recorder := NewRecorder(session.ID, store)
	defer recorder.Stop()

	proxy, err := NewProxy(target.URL, recorder, ProxyOptions{MaxBodySize: 4})
	if err != nil {
		t.Fatalf("NewProxy() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://proxy.local/api/test", strings.NewReader("123456"))
	resp := httptest.NewRecorder()
	proxy.ServeHTTP(resp, req)

	records, err := store.ListRecords(session.ID)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	record := records[0]
	if got := string(record.Request.Body); got != "1234" {
		t.Fatalf("request body = %q, want %q", got, "1234")
	}
	if record.Request.BodyLen != 6 {
		t.Fatalf("request bodyLen = %d, want %d", record.Request.BodyLen, 6)
	}
	if upstreamBody != "123456" {
		t.Fatalf("upstream body = %q, want %q", upstreamBody, "123456")
	}
	if got := string(record.Response.Body); got != "resp" {
		t.Fatalf("response body = %q, want %q", got, "resp")
	}
	if record.Response.BodyLen != 11 {
		t.Fatalf("response bodyLen = %d, want %d", record.Response.BodyLen, 11)
	}
}

func TestProxy_LargeRequestBodyIsFullyForwarded(t *testing.T) {
	const requestBody = "abcdefghijklmnopqrstuvwxyz"

	var upstreamBody string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		upstreamBody = string(body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer target.Close()

	store, session := newProxyTestStore(t)
	recorder := NewRecorder(session.ID, store)
	defer recorder.Stop()

	proxy, err := NewProxy(target.URL, recorder, ProxyOptions{MaxBodySize: 5})
	if err != nil {
		t.Fatalf("NewProxy() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://proxy.local/api/upload", strings.NewReader(requestBody))
	resp := httptest.NewRecorder()
	proxy.ServeHTTP(resp, req)

	records, err := store.ListRecords(session.ID)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if upstreamBody != requestBody {
		t.Fatalf("upstream body = %q, want %q", upstreamBody, requestBody)
	}
	if got := string(records[0].Request.Body); got != requestBody[:5] {
		t.Fatalf("request body = %q, want %q", got, requestBody[:5])
	}
	if records[0].Request.BodyLen != int64(len(requestBody)) {
		t.Fatalf("request bodyLen = %d, want %d", records[0].Request.BodyLen, len(requestBody))
	}
}

func TestProxy_ExcludePathSkipsRecording(t *testing.T) {
	const requestBody = "skip-me-but-forward-me"

	var upstreamBody string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		upstreamBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	store, session := newProxyTestStore(t)
	recorder := NewRecorder(session.ID, store)
	defer recorder.Stop()

	proxy, err := NewProxy(target.URL, recorder, ProxyOptions{ExcludePathPrefixes: []string{"/health"}})
	if err != nil {
		t.Fatalf("NewProxy() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://proxy.local/healthz", strings.NewReader(requestBody))
	resp := httptest.NewRecorder()
	proxy.ServeHTTP(resp, req)

	records, err := store.ListRecords(session.ID)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
	if upstreamBody != requestBody {
		t.Fatalf("upstream body = %q, want %q", upstreamBody, requestBody)
	}
}
