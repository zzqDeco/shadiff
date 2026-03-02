package model

import (
	"testing"
)

// --- Session ---

func TestSessionStatus_Constants(t *testing.T) {
	tests := []struct {
		name   string
		status SessionStatus
		want   string
	}{
		{"recording", SessionRecording, "recording"},
		{"completed", SessionCompleted, "completed"},
		{"replayed", SessionReplayed, "replayed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("SessionStatus = %q, want %q", tt.status, tt.want)
			}
		})
	}
}

func TestSession_FieldAssignment(t *testing.T) {
	s := Session{
		ID:          "abcd1234",
		Name:        "test-session",
		Description: "A test session",
		Source: EndpointConfig{
			BaseURL: "http://localhost:8080",
			Headers: map[string]string{"X-Source": "true"},
		},
		Target: EndpointConfig{
			BaseURL: "http://localhost:9090",
			Headers: map[string]string{"X-Target": "true"},
		},
		Tags:        []string{"tag1", "tag2"},
		RecordCount: 5,
		CreatedAt:   1700000000000,
		UpdatedAt:   1700000001000,
		Status:      SessionRecording,
		Metadata:    map[string]string{"env": "test"},
	}

	if s.ID != "abcd1234" {
		t.Errorf("ID = %q, want %q", s.ID, "abcd1234")
	}
	if s.Name != "test-session" {
		t.Errorf("Name = %q, want %q", s.Name, "test-session")
	}
	if s.Description != "A test session" {
		t.Errorf("Description = %q, want %q", s.Description, "A test session")
	}
	if s.Source.BaseURL != "http://localhost:8080" {
		t.Errorf("Source.BaseURL = %q, want %q", s.Source.BaseURL, "http://localhost:8080")
	}
	if s.Source.Headers["X-Source"] != "true" {
		t.Errorf("Source.Headers[X-Source] = %q, want %q", s.Source.Headers["X-Source"], "true")
	}
	if s.Target.BaseURL != "http://localhost:9090" {
		t.Errorf("Target.BaseURL = %q, want %q", s.Target.BaseURL, "http://localhost:9090")
	}
	if len(s.Tags) != 2 || s.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want [tag1 tag2]", s.Tags)
	}
	if s.RecordCount != 5 {
		t.Errorf("RecordCount = %d, want 5", s.RecordCount)
	}
	if s.CreatedAt != 1700000000000 {
		t.Errorf("CreatedAt = %d, want 1700000000000", s.CreatedAt)
	}
	if s.UpdatedAt != 1700000001000 {
		t.Errorf("UpdatedAt = %d, want 1700000001000", s.UpdatedAt)
	}
	if s.Status != SessionRecording {
		t.Errorf("Status = %q, want %q", s.Status, SessionRecording)
	}
	if s.Metadata["env"] != "test" {
		t.Errorf("Metadata[env] = %q, want %q", s.Metadata["env"], "test")
	}
}

func TestEndpointConfig_FieldAssignment(t *testing.T) {
	ec := EndpointConfig{
		BaseURL: "http://example.com",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}
	if ec.BaseURL != "http://example.com" {
		t.Errorf("BaseURL = %q, want %q", ec.BaseURL, "http://example.com")
	}
	if ec.Headers["Authorization"] != "Bearer token" {
		t.Errorf("Headers[Authorization] = %q, want %q", ec.Headers["Authorization"], "Bearer token")
	}
}

func TestSessionFilter_FieldAssignment(t *testing.T) {
	f := SessionFilter{
		Name:   "search",
		Status: "completed",
		Tags:   []string{"api"},
	}
	if f.Name != "search" {
		t.Errorf("Name = %q, want %q", f.Name, "search")
	}
	if f.Status != "completed" {
		t.Errorf("Status = %q, want %q", f.Status, "completed")
	}
	if len(f.Tags) != 1 || f.Tags[0] != "api" {
		t.Errorf("Tags = %v, want [api]", f.Tags)
	}
}

// --- Record ---

func TestRecord_FieldAssignment(t *testing.T) {
	r := Record{
		ID:        "rec-001",
		SessionID: "sess-001",
		Sequence:  1,
		Request: HTTPRequest{
			Method: "GET",
			Path:   "/api/v1/users",
		},
		Response: HTTPResponse{
			StatusCode: 200,
		},
		SideEffects: []SideEffect{
			{Type: SideEffectDB, DBType: "mysql"},
		},
		Duration:   150,
		RecordedAt: 1700000000000,
		Error:      "timeout",
	}

	if r.ID != "rec-001" {
		t.Errorf("ID = %q, want %q", r.ID, "rec-001")
	}
	if r.SessionID != "sess-001" {
		t.Errorf("SessionID = %q, want %q", r.SessionID, "sess-001")
	}
	if r.Sequence != 1 {
		t.Errorf("Sequence = %d, want 1", r.Sequence)
	}
	if r.Request.Method != "GET" {
		t.Errorf("Request.Method = %q, want %q", r.Request.Method, "GET")
	}
	if r.Response.StatusCode != 200 {
		t.Errorf("Response.StatusCode = %d, want 200", r.Response.StatusCode)
	}
	if len(r.SideEffects) != 1 {
		t.Fatalf("len(SideEffects) = %d, want 1", len(r.SideEffects))
	}
	if r.SideEffects[0].Type != SideEffectDB {
		t.Errorf("SideEffects[0].Type = %q, want %q", r.SideEffects[0].Type, SideEffectDB)
	}
	if r.Duration != 150 {
		t.Errorf("Duration = %d, want 150", r.Duration)
	}
	if r.RecordedAt != 1700000000000 {
		t.Errorf("RecordedAt = %d, want 1700000000000", r.RecordedAt)
	}
	if r.Error != "timeout" {
		t.Errorf("Error = %q, want %q", r.Error, "timeout")
	}
}

// --- HTTPRequest / HTTPResponse ---

func TestHTTPRequest_FieldAssignment(t *testing.T) {
	req := HTTPRequest{
		Method:  "POST",
		Path:    "/api/data",
		Query:   "page=1&size=10",
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    []byte(`{"key":"value"}`),
		BodyLen: 15,
	}
	if req.Method != "POST" {
		t.Errorf("Method = %q, want %q", req.Method, "POST")
	}
	if req.Path != "/api/data" {
		t.Errorf("Path = %q, want %q", req.Path, "/api/data")
	}
	if req.Query != "page=1&size=10" {
		t.Errorf("Query = %q, want %q", req.Query, "page=1&size=10")
	}
	if len(req.Headers["Content-Type"]) != 1 || req.Headers["Content-Type"][0] != "application/json" {
		t.Errorf("Headers[Content-Type] = %v, want [application/json]", req.Headers["Content-Type"])
	}
	if string(req.Body) != `{"key":"value"}` {
		t.Errorf("Body = %q, want %q", string(req.Body), `{"key":"value"}`)
	}
	if req.BodyLen != 15 {
		t.Errorf("BodyLen = %d, want 15", req.BodyLen)
	}
}

func TestHTTPResponse_FieldAssignment(t *testing.T) {
	resp := HTTPResponse{
		StatusCode: 404,
		Headers:    map[string][]string{"X-Custom": {"val1", "val2"}},
		Body:       []byte("not found"),
		BodyLen:    9,
	}
	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	if len(resp.Headers["X-Custom"]) != 2 {
		t.Errorf("len(Headers[X-Custom]) = %d, want 2", len(resp.Headers["X-Custom"]))
	}
	if string(resp.Body) != "not found" {
		t.Errorf("Body = %q, want %q", string(resp.Body), "not found")
	}
	if resp.BodyLen != 9 {
		t.Errorf("BodyLen = %d, want 9", resp.BodyLen)
	}
}

// --- SideEffect ---

func TestSideEffectType_Constants(t *testing.T) {
	tests := []struct {
		name string
		typ  SideEffectType
		want string
	}{
		{"database", SideEffectDB, "database"},
		{"http_call", SideEffectHTTP, "http_call"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.typ) != tt.want {
				t.Errorf("SideEffectType = %q, want %q", tt.typ, tt.want)
			}
		})
	}
}

func TestSideEffect_DBFields(t *testing.T) {
	se := SideEffect{
		Type:      SideEffectDB,
		Timestamp: 1700000000000,
		Duration:  25,
		DBType:    "postgres",
		Query:     "SELECT * FROM users WHERE id = $1",
		Args:      []any{42},
		RowCount:  1,
	}
	if se.Type != SideEffectDB {
		t.Errorf("Type = %q, want %q", se.Type, SideEffectDB)
	}
	if se.Timestamp != 1700000000000 {
		t.Errorf("Timestamp = %d, want 1700000000000", se.Timestamp)
	}
	if se.Duration != 25 {
		t.Errorf("Duration = %d, want 25", se.Duration)
	}
	if se.DBType != "postgres" {
		t.Errorf("DBType = %q, want %q", se.DBType, "postgres")
	}
	if se.Query != "SELECT * FROM users WHERE id = $1" {
		t.Errorf("Query = %q, want expected", se.Query)
	}
	if len(se.Args) != 1 {
		t.Fatalf("len(Args) = %d, want 1", len(se.Args))
	}
	if se.RowCount != 1 {
		t.Errorf("RowCount = %d, want 1", se.RowCount)
	}
}

func TestSideEffect_MongoFields(t *testing.T) {
	se := SideEffect{
		Type:       SideEffectDB,
		DBType:     "mongo",
		Database:   "testdb",
		Collection: "users",
		Operation:  "find",
		Filter:     map[string]any{"name": "alice"},
		DocCount:   3,
	}
	if se.Database != "testdb" {
		t.Errorf("Database = %q, want %q", se.Database, "testdb")
	}
	if se.Collection != "users" {
		t.Errorf("Collection = %q, want %q", se.Collection, "users")
	}
	if se.Operation != "find" {
		t.Errorf("Operation = %q, want %q", se.Operation, "find")
	}
	if se.Filter == nil {
		t.Error("Filter should not be nil")
	}
	if se.DocCount != 3 {
		t.Errorf("DocCount = %d, want 3", se.DocCount)
	}
}

func TestSideEffect_HTTPFields(t *testing.T) {
	req := &HTTPRequest{Method: "GET", Path: "/external"}
	resp := &HTTPResponse{StatusCode: 200}
	se := SideEffect{
		Type:     SideEffectHTTP,
		HTTPReq:  req,
		HTTPResp: resp,
	}
	if se.HTTPReq == nil {
		t.Fatal("HTTPReq should not be nil")
	}
	if se.HTTPReq.Method != "GET" {
		t.Errorf("HTTPReq.Method = %q, want %q", se.HTTPReq.Method, "GET")
	}
	if se.HTTPResp == nil {
		t.Fatal("HTTPResp should not be nil")
	}
	if se.HTTPResp.StatusCode != 200 {
		t.Errorf("HTTPResp.StatusCode = %d, want 200", se.HTTPResp.StatusCode)
	}
}

// --- Diff ---

func TestDifferenceKind_Constants(t *testing.T) {
	tests := []struct {
		name string
		kind DifferenceKind
		want string
	}{
		{"status_code", DiffStatusCode, "status_code"},
		{"header", DiffHeader, "header"},
		{"body", DiffBody, "body"},
		{"body_field", DiffBodyField, "body_field"},
		{"db_query", DiffDBQuery, "db_query"},
		{"db_query_count", DiffDBQueryCount, "db_query_count"},
		{"mongo_op", DiffMongoOp, "mongo_op"},
		{"external_call", DiffExternalCall, "external_call"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.kind) != tt.want {
				t.Errorf("DifferenceKind = %q, want %q", tt.kind, tt.want)
			}
		})
	}
}

func TestSeverity_Constants(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{"error", SeverityError, "error"},
		{"warning", SeverityWarning, "warning"},
		{"info", SeverityInfo, "info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.severity) != tt.want {
				t.Errorf("Severity = %q, want %q", tt.severity, tt.want)
			}
		})
	}
}

func TestDifference_FieldAssignment(t *testing.T) {
	d := Difference{
		Kind:     DiffBody,
		Path:     "body.data.items[0].name",
		Expected: "alice",
		Actual:   "bob",
		Message:  "name mismatch",
		Severity: SeverityError,
		Ignored:  true,
		Rule:     "ignore-name",
	}
	if d.Kind != DiffBody {
		t.Errorf("Kind = %q, want %q", d.Kind, DiffBody)
	}
	if d.Path != "body.data.items[0].name" {
		t.Errorf("Path = %q, want expected", d.Path)
	}
	if d.Expected != "alice" {
		t.Errorf("Expected = %v, want alice", d.Expected)
	}
	if d.Actual != "bob" {
		t.Errorf("Actual = %v, want bob", d.Actual)
	}
	if d.Message != "name mismatch" {
		t.Errorf("Message = %q, want %q", d.Message, "name mismatch")
	}
	if d.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q", d.Severity, SeverityError)
	}
	if !d.Ignored {
		t.Error("Ignored should be true")
	}
	if d.Rule != "ignore-name" {
		t.Errorf("Rule = %q, want %q", d.Rule, "ignore-name")
	}
}

func TestDiffResult_FieldAssignment(t *testing.T) {
	dr := DiffResult{
		RecordID: "rec-001",
		Sequence: 1,
		Request:  HTTPRequest{Method: "GET", Path: "/api/test"},
		Match:    false,
		Differences: []Difference{
			{Kind: DiffStatusCode, Severity: SeverityError},
			{Kind: DiffBody, Severity: SeverityWarning},
		},
	}
	if dr.RecordID != "rec-001" {
		t.Errorf("RecordID = %q, want %q", dr.RecordID, "rec-001")
	}
	if dr.Sequence != 1 {
		t.Errorf("Sequence = %d, want 1", dr.Sequence)
	}
	if dr.Request.Method != "GET" {
		t.Errorf("Request.Method = %q, want %q", dr.Request.Method, "GET")
	}
	if dr.Match {
		t.Error("Match should be false")
	}
	if len(dr.Differences) != 2 {
		t.Fatalf("len(Differences) = %d, want 2", len(dr.Differences))
	}
	if dr.Differences[0].Kind != DiffStatusCode {
		t.Errorf("Differences[0].Kind = %q, want %q", dr.Differences[0].Kind, DiffStatusCode)
	}
}

func TestDiffResult_MatchTrue(t *testing.T) {
	dr := DiffResult{
		RecordID:    "rec-002",
		Sequence:    2,
		Match:       true,
		Differences: nil,
	}
	if !dr.Match {
		t.Error("Match should be true")
	}
	if dr.Differences != nil {
		t.Errorf("Differences should be nil, got %v", dr.Differences)
	}
}

func TestDiffSummary_FieldAssignment(t *testing.T) {
	ds := DiffSummary{
		SessionID:   "sess-001",
		TotalCount:  10,
		MatchCount:  7,
		DiffCount:   3,
		ErrorCount:  1,
		IgnoreCount: 2,
		MatchRate:   0.7,
	}
	if ds.SessionID != "sess-001" {
		t.Errorf("SessionID = %q, want %q", ds.SessionID, "sess-001")
	}
	if ds.TotalCount != 10 {
		t.Errorf("TotalCount = %d, want 10", ds.TotalCount)
	}
	if ds.MatchCount != 7 {
		t.Errorf("MatchCount = %d, want 7", ds.MatchCount)
	}
	if ds.DiffCount != 3 {
		t.Errorf("DiffCount = %d, want 3", ds.DiffCount)
	}
	if ds.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", ds.ErrorCount)
	}
	if ds.IgnoreCount != 2 {
		t.Errorf("IgnoreCount = %d, want 2", ds.IgnoreCount)
	}
	if ds.MatchRate != 0.7 {
		t.Errorf("MatchRate = %f, want 0.7", ds.MatchRate)
	}
}

// --- Zero value tests ---

func TestSession_ZeroValue(t *testing.T) {
	var s Session
	if s.ID != "" {
		t.Errorf("zero ID = %q, want empty", s.ID)
	}
	if s.RecordCount != 0 {
		t.Errorf("zero RecordCount = %d, want 0", s.RecordCount)
	}
	if s.Status != "" {
		t.Errorf("zero Status = %q, want empty", s.Status)
	}
	if s.Tags != nil {
		t.Errorf("zero Tags = %v, want nil", s.Tags)
	}
	if s.Metadata != nil {
		t.Errorf("zero Metadata = %v, want nil", s.Metadata)
	}
}

func TestRecord_ZeroValue(t *testing.T) {
	var r Record
	if r.ID != "" {
		t.Errorf("zero ID = %q, want empty", r.ID)
	}
	if r.SideEffects != nil {
		t.Errorf("zero SideEffects = %v, want nil", r.SideEffects)
	}
	if r.Duration != 0 {
		t.Errorf("zero Duration = %d, want 0", r.Duration)
	}
}

func TestSideEffect_NilPointers(t *testing.T) {
	se := SideEffect{Type: SideEffectDB}
	if se.HTTPReq != nil {
		t.Error("HTTPReq should be nil for DB side effect")
	}
	if se.HTTPResp != nil {
		t.Error("HTTPResp should be nil for DB side effect")
	}
}
