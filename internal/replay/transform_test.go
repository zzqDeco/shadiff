package replay

import (
	"io"
	"net/http"
	"testing"

	"shadiff/internal/model"
)

func TestTransform_BasicRequest(t *testing.T) {
	req := model.HTTPRequest{
		Method: "POST",
		Path:   "/api/users",
		Query:  "page=1&limit=10",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
			"Accept":       {"application/json"},
		},
		Body: []byte(`{"name":"alice"}`),
	}
	cfg := TransformConfig{
		TargetBaseURL: "http://localhost:8080",
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	// Method
	if result.Method != "POST" {
		t.Errorf("method: got %q, want %q", result.Method, "POST")
	}

	// URL path
	if result.URL.Path != "/api/users" {
		t.Errorf("path: got %q, want %q", result.URL.Path, "/api/users")
	}

	// Query string
	if result.URL.RawQuery != "page=1&limit=10" {
		t.Errorf("query: got %q, want %q", result.URL.RawQuery, "page=1&limit=10")
	}

	// Host
	if result.URL.Host != "localhost:8080" {
		t.Errorf("host: got %q, want %q", result.URL.Host, "localhost:8080")
	}

	// Headers copied
	if got := result.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type header: got %q, want %q", got, "application/json")
	}
	if got := result.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept header: got %q, want %q", got, "application/json")
	}

	// Body
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != `{"name":"alice"}` {
		t.Errorf("body: got %q, want %q", string(body), `{"name":"alice"}`)
	}
}

func TestTransform_NoQuery(t *testing.T) {
	req := model.HTTPRequest{
		Method: "GET",
		Path:   "/health",
	}
	cfg := TransformConfig{
		TargetBaseURL: "http://example.com",
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	if result.URL.RawQuery != "" {
		t.Errorf("query: got %q, want empty", result.URL.RawQuery)
	}
	if result.URL.String() != "http://example.com/health" {
		t.Errorf("url: got %q, want %q", result.URL.String(), "http://example.com/health")
	}
}

func TestTransform_HostSubstitution(t *testing.T) {
	req := model.HTTPRequest{
		Method: "GET",
		Path:   "/api/data",
	}
	cfg := TransformConfig{
		TargetBaseURL: "https://staging.example.com:9443",
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	if result.URL.Scheme != "https" {
		t.Errorf("scheme: got %q, want %q", result.URL.Scheme, "https")
	}
	if result.URL.Host != "staging.example.com:9443" {
		t.Errorf("host: got %q, want %q", result.URL.Host, "staging.example.com:9443")
	}
}

func TestTransform_HeaderOverrides(t *testing.T) {
	req := model.HTTPRequest{
		Method: "GET",
		Path:   "/test",
		Headers: map[string][]string{
			"Authorization": {"Bearer old-token"},
			"Accept":        {"text/plain"},
		},
	}
	cfg := TransformConfig{
		TargetBaseURL: "http://localhost",
		HeaderOverride: map[string]string{
			"Authorization": "Bearer new-token",
			"X-Custom":      "added",
		},
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	// Overridden header
	if got := result.Header.Get("Authorization"); got != "Bearer new-token" {
		t.Errorf("Authorization: got %q, want %q", got, "Bearer new-token")
	}

	// New header added via override
	if got := result.Header.Get("X-Custom"); got != "added" {
		t.Errorf("X-Custom: got %q, want %q", got, "added")
	}

	// Original header preserved
	if got := result.Header.Get("Accept"); got != "text/plain" {
		t.Errorf("Accept: got %q, want %q", got, "text/plain")
	}
}

func TestTransform_HeaderRemoval(t *testing.T) {
	req := model.HTTPRequest{
		Method: "GET",
		Path:   "/test",
		Headers: map[string][]string{
			"Authorization": {"Bearer token"},
			"Cookie":        {"session=abc"},
			"Accept":        {"*/*"},
		},
	}
	cfg := TransformConfig{
		TargetBaseURL: "http://localhost",
		HeaderRemove:  []string{"Cookie", "Authorization"},
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	if got := result.Header.Get("Cookie"); got != "" {
		t.Errorf("Cookie should be removed, got %q", got)
	}
	if got := result.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization should be removed, got %q", got)
	}
	// Accept should remain
	if got := result.Header.Get("Accept"); got != "*/*" {
		t.Errorf("Accept: got %q, want %q", got, "*/*")
	}
}

func TestTransform_ProxyHeaderRemoval(t *testing.T) {
	proxyHeaders := []string{"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"}

	req := model.HTTPRequest{
		Method:  "GET",
		Path:    "/test",
		Headers: make(map[string][]string),
	}
	for _, h := range proxyHeaders {
		req.Headers[h] = []string{"some-value"}
	}

	cfg := TransformConfig{
		TargetBaseURL: "http://localhost",
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	for _, h := range proxyHeaders {
		if got := result.Header.Get(h); got != "" {
			t.Errorf("proxy header %s should be removed, got %q", h, got)
		}
	}
}

func TestTransform_HeaderRemoveBeforeOverride(t *testing.T) {
	// Verify that removal happens before override, so an override can re-add a removed header
	req := model.HTTPRequest{
		Method: "GET",
		Path:   "/test",
		Headers: map[string][]string{
			"X-Target": {"original"},
		},
	}
	cfg := TransformConfig{
		TargetBaseURL:  "http://localhost",
		HeaderRemove:   []string{"X-Target"},
		HeaderOverride: map[string]string{"X-Target": "overridden"},
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	// The code removes first, then overrides, so the override should win
	if got := result.Header.Get("X-Target"); got != "overridden" {
		t.Errorf("X-Target: got %q, want %q", got, "overridden")
	}
}

func TestTransform_EmptyBody(t *testing.T) {
	req := model.HTTPRequest{
		Method: "GET",
		Path:   "/",
	}
	cfg := TransformConfig{
		TargetBaseURL: "http://localhost",
	}

	result := Transform(req, cfg)
	if result == nil {
		t.Fatal("expected non-nil request")
	}

	if result.Method != http.MethodGet {
		t.Errorf("method: got %q, want %q", result.Method, http.MethodGet)
	}
}
