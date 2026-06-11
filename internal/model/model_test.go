package model

import (
	"encoding/json"
	"strings"
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
			NewSQLSideEffect("mysql", "SELECT 1", 1700000000000),
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
		BodyRef: "artifacts/request-bodies/rec-1.bin",
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
	if req.BodyRef != "artifacts/request-bodies/rec-1.bin" {
		t.Errorf("BodyRef = %q, want %q", req.BodyRef, "artifacts/request-bodies/rec-1.bin")
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
		Database: &DatabaseSideEffect{
			Type: "postgres",
			SQL: &SQLSideEffect{
				Query:    "SELECT * FROM users WHERE id = $1",
				Args:     []any{42},
				RowCount: 1,
			},
		},
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
	if se.DatabaseType() != "postgres" {
		t.Errorf("DatabaseType() = %q, want %q", se.DatabaseType(), "postgres")
	}
	if se.SQL().Query != "SELECT * FROM users WHERE id = $1" {
		t.Errorf("SQL.Query = %q, want expected", se.SQL().Query)
	}
	if len(se.SQL().Args) != 1 {
		t.Fatalf("len(SQL.Args) = %d, want 1", len(se.SQL().Args))
	}
	if se.SQL().RowCount != 1 {
		t.Errorf("SQL.RowCount = %d, want 1", se.SQL().RowCount)
	}
}

func TestSideEffect_MongoFields(t *testing.T) {
	se := SideEffect{
		Type: SideEffectDB,
		Database: &DatabaseSideEffect{
			Type: "mongo",
			Mongo: &MongoSideEffect{
				Database:   "testdb",
				Collection: "users",
				Operation:  "find",
				Filter:     map[string]any{"name": "alice"},
				DocCount:   3,
			},
		},
	}
	if se.Mongo().Database != "testdb" {
		t.Errorf("Mongo.Database = %q, want %q", se.Mongo().Database, "testdb")
	}
	if se.Mongo().Collection != "users" {
		t.Errorf("Mongo.Collection = %q, want %q", se.Mongo().Collection, "users")
	}
	if se.Mongo().Operation != "find" {
		t.Errorf("Mongo.Operation = %q, want %q", se.Mongo().Operation, "find")
	}
	if se.Mongo().Filter == nil {
		t.Error("Filter should not be nil")
	}
	if se.Mongo().DocCount != 3 {
		t.Errorf("Mongo.DocCount = %d, want 3", se.Mongo().DocCount)
	}
}

func TestSideEffect_RedisFields(t *testing.T) {
	se := NewRedisSideEffect("SET", "user:1", []string{"user:1", "ada"}, 1700000000000)
	if se.Redis().Command != "SET" {
		t.Errorf("Redis.Command = %q, want SET", se.Redis().Command)
	}
	if se.Redis().Key != "user:1" {
		t.Errorf("Redis.Key = %q, want user:1", se.Redis().Key)
	}
	if len(se.Redis().Args) != 2 || se.Redis().Args[0] != "user:1" || se.Redis().Args[1] != "ada" {
		t.Errorf("Redis.Args = %+v, want [user:1 ada]", se.Redis().Args)
	}
}

func TestSideEffect_HTTPFields(t *testing.T) {
	req := &HTTPRequest{Method: "GET", Path: "/external"}
	resp := &HTTPResponse{StatusCode: 200}
	se := SideEffect{
		Type: SideEffectHTTP,
		HTTP: &HTTPSideEffect{
			Request:  req,
			Response: resp,
		},
	}
	if se.HTTP == nil || se.HTTP.Request == nil {
		t.Fatal("HTTP.Request should not be nil")
	}
	if se.HTTP.Request.Method != "GET" {
		t.Errorf("HTTP.Request.Method = %q, want %q", se.HTTP.Request.Method, "GET")
	}
	if se.HTTP.Response == nil {
		t.Fatal("HTTP.Response should not be nil")
	}
	if se.HTTP.Response.StatusCode != 200 {
		t.Errorf("HTTP.Response.StatusCode = %d, want 200", se.HTTP.Response.StatusCode)
	}
}

func TestSideEffect_JSONUsesTypedPayload(t *testing.T) {
	se := NewSQLSideEffect("mysql", "SELECT 1", 1700000000000)
	data, err := json.Marshal(se)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	got := string(data)
	var topLevel map[string]any
	if err := json.Unmarshal(data, &topLevel); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	for _, removedField := range []string{"dbType", "query", "redisCommand", "collection"} {
		if _, ok := topLevel[removedField]; ok {
			t.Fatalf("JSON %s contains removed top-level field %q", got, removedField)
		}
	}
	if !strings.Contains(got, `"database":{"type":"mysql","sql":{"query":"SELECT 1"}}`) {
		t.Fatalf("JSON = %s, want typed database payload", got)
	}
}

func TestRecordSideEffects_JSONUsesTypedDatabasePayloads(t *testing.T) {
	record := Record{
		ID:        "rec-typed-sideeffects",
		SessionID: "session-typed-sideeffects",
		Sequence:  1,
		SideEffects: []SideEffect{
			NewSQLSideEffect("mysql", "SELECT 1", 1700000000000),
			NewSQLSideEffect("postgres", "SELECT 2", 1700000000001),
			NewMongoSideEffect(MongoSideEffect{
				Database:   "app",
				Collection: "users",
				Operation:  "find",
			}, 1700000000002),
			NewRedisSideEffect("SET", "user:1", []string{"user:1", "ada"}, 1700000000003),
		},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	var parsed struct {
		SideEffects []map[string]any `json:"sideEffects"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if len(parsed.SideEffects) != 4 {
		t.Fatalf("sideEffects count = %d, want 4", len(parsed.SideEffects))
	}

	assertNoRemovedTopLevelSideEffectFields := func(t *testing.T, effect map[string]any) {
		t.Helper()
		for _, removedField := range []string{"dbType", "query", "collection", "redisCommand", "redisKey", "redisArgs"} {
			if _, ok := effect[removedField]; ok {
				t.Fatalf("side effect JSON contains removed top-level field %q: %s", removedField, string(data))
			}
		}
	}
	assertDatabasePayload := func(t *testing.T, index int, dbType, payloadField string) map[string]any {
		t.Helper()
		effect := parsed.SideEffects[index]
		assertNoRemovedTopLevelSideEffectFields(t, effect)
		if effect["type"] != string(SideEffectDB) {
			t.Fatalf("sideEffects[%d].type = %v, want %q", index, effect["type"], SideEffectDB)
		}
		database, ok := effect["database"].(map[string]any)
		if !ok {
			t.Fatalf("sideEffects[%d].database missing or wrong type: %T", index, effect["database"])
		}
		if database["type"] != dbType {
			t.Fatalf("sideEffects[%d].database.type = %v, want %q", index, database["type"], dbType)
		}
		if _, ok := database[payloadField]; !ok {
			t.Fatalf("sideEffects[%d].database missing payload %q: %#v", index, payloadField, database)
		}
		return database
	}

	mysql := assertDatabasePayload(t, 0, "mysql", "sql")
	if mysql["sql"].(map[string]any)["query"] != "SELECT 1" {
		t.Fatalf("mysql sql query = %v, want SELECT 1", mysql["sql"].(map[string]any)["query"])
	}
	postgres := assertDatabasePayload(t, 1, "postgres", "sql")
	if postgres["sql"].(map[string]any)["query"] != "SELECT 2" {
		t.Fatalf("postgres sql query = %v, want SELECT 2", postgres["sql"].(map[string]any)["query"])
	}
	mongo := assertDatabasePayload(t, 2, "mongo", "mongo")
	if mongo["mongo"].(map[string]any)["collection"] != "users" {
		t.Fatalf("mongo collection = %v, want users", mongo["mongo"].(map[string]any)["collection"])
	}
	redis := assertDatabasePayload(t, 3, "redis", "redis")
	if redis["redis"].(map[string]any)["command"] != "SET" {
		t.Fatalf("redis command = %v, want SET", redis["redis"].(map[string]any)["command"])
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
		{"redis_command", DiffRedisCommand, "redis_command"},
		{"redis_command_count", DiffRedisCount, "redis_command_count"},
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
	if se.HTTP != nil {
		t.Error("HTTP should be nil for DB side effect")
	}
	if se.Database != nil {
		t.Error("Database should be nil for zero DB side effect")
	}
}
