package diff

import (
	"testing"

	"shadiff/internal/model"
)

func TestDefaultRules(t *testing.T) {
	rs := DefaultRules()
	if rs == nil {
		t.Fatal("DefaultRules() returned nil")
	}
	if len(rs.Rules) == 0 {
		t.Error("DefaultRules() returned empty rules")
	}
	// Should have timestamp and uuid matchers registered
	if _, ok := rs.matchers["timestamp"]; !ok {
		t.Error("expected timestamp matcher to be registered")
	}
	if _, ok := rs.matchers["uuid"]; !ok {
		t.Error("expected uuid matcher to be registered")
	}
	if _, ok := rs.matchers["numeric_tolerance"]; !ok {
		t.Error("expected numeric_tolerance matcher to be registered")
	}
}

func TestTimestampMatcher(t *testing.T) {
	m := TimestampMatcher{}
	if m.Name() != "timestamp" {
		t.Errorf("expected name 'timestamp', got %q", m.Name())
	}

	tests := []struct {
		name     string
		expected any
		actual   any
		want     bool
	}{
		{"both ISO timestamps", "2024-01-15T10:30:00Z", "2024-02-20T14:00:00Z", true},
		{"both space-separated timestamps", "2024-01-15 10:30:00", "2024-02-20 14:00:00", true},
		{"one not timestamp", "2024-01-15T10:30:00Z", "not-a-timestamp", false},
		{"neither timestamps", "hello", "world", false},
		{"non-string expected", 12345, "2024-01-15T10:30:00Z", false},
		{"non-string actual", "2024-01-15T10:30:00Z", 12345, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Match("some.path", tt.expected, tt.actual)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Match(%v, %v) = %v, want %v", tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestUUIDMatcher(t *testing.T) {
	m := UUIDMatcher{}
	if m.Name() != "uuid" {
		t.Errorf("expected name 'uuid', got %q", m.Name())
	}

	tests := []struct {
		name     string
		expected any
		actual   any
		want     bool
	}{
		{"both UUIDs", "550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"one not UUID", "550e8400-e29b-41d4-a716-446655440000", "not-a-uuid", false},
		{"neither UUIDs", "hello", "world", false},
		{"non-string values", 123, 456, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Match("some.path", tt.expected, tt.actual)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Match(%v, %v) = %v, want %v", tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestNumericToleranceMatcher(t *testing.T) {
	m := NumericToleranceMatcher{Tolerance: 0.01}
	if m.Name() != "numeric_tolerance" {
		t.Errorf("expected name 'numeric_tolerance', got %q", m.Name())
	}

	tests := []struct {
		name     string
		expected any
		actual   any
		want     bool
	}{
		{"within tolerance", 1.005, 1.010, true},
		{"exactly at tolerance", 1.0, 1.009, true},
		{"beyond tolerance", 1.0, 1.02, false},
		{"identical values", 5.0, 5.0, true},
		{"non-numeric values", "abc", "def", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Match("some.path", tt.expected, tt.actual)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Match(%v, %v) = %v, want %v", tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestRuleSet_Apply_IgnoreRule(t *testing.T) {
	rs := NewRuleSet([]Rule{
		{
			Name:  "ignore_id",
			Kind:  "ignore",
			Paths: []string{"body.id"},
		},
	})

	diffs := []model.Difference{
		{Kind: model.DiffBodyField, Path: "body.id", Expected: "a", Actual: "b"},
		{Kind: model.DiffBodyField, Path: "body.name", Expected: "x", Actual: "y"},
	}

	result := rs.Apply(diffs)

	if !result[0].Ignored {
		t.Error("expected body.id diff to be marked as ignored")
	}
	if result[0].Rule != "ignore_id" {
		t.Errorf("expected rule name 'ignore_id', got %q", result[0].Rule)
	}
	if result[1].Ignored {
		t.Error("expected body.name diff to NOT be ignored")
	}
}

func TestRuleSet_Apply_CustomMatcher(t *testing.T) {
	rs := NewRuleSet(
		[]Rule{
			{
				Name:    "timestamp_rule",
				Kind:    "custom",
				Paths:   []string{"body.createdAt"},
				Matcher: "timestamp",
			},
		},
		TimestampMatcher{},
	)

	diffs := []model.Difference{
		{
			Kind:     model.DiffBodyField,
			Path:     "body.createdAt",
			Expected: "2024-01-15T10:00:00Z",
			Actual:   "2024-02-20T14:00:00Z",
		},
	}

	result := rs.Apply(diffs)
	if !result[0].Ignored {
		t.Error("expected timestamp diff to be marked as ignored by custom matcher")
	}
	if result[0].Rule != "timestamp_rule" {
		t.Errorf("expected rule 'timestamp_rule', got %q", result[0].Rule)
	}
}

func TestRuleSet_Apply_CustomMatcher_NoMatch(t *testing.T) {
	rs := NewRuleSet(
		[]Rule{
			{
				Name:    "timestamp_rule",
				Kind:    "custom",
				Paths:   []string{"body.createdAt"},
				Matcher: "timestamp",
			},
		},
		TimestampMatcher{},
	)

	// Non-timestamp values should not match
	diffs := []model.Difference{
		{
			Kind:     model.DiffBodyField,
			Path:     "body.createdAt",
			Expected: "not-a-timestamp",
			Actual:   "also-not-a-timestamp",
		},
	}

	result := rs.Apply(diffs)
	if result[0].Ignored {
		t.Error("expected non-timestamp values to NOT be ignored")
	}
}

func TestPathPattern_Wildcard_SingleLevel(t *testing.T) {
	rs := NewRuleSet([]Rule{
		{
			Name:  "ignore_any_id",
			Kind:  "ignore",
			Paths: []string{"body.*.id"},
		},
	})

	diffs := []model.Difference{
		{Kind: model.DiffBodyField, Path: "body.user.id", Expected: "a", Actual: "b"},
		{Kind: model.DiffBodyField, Path: "body.order.id", Expected: "c", Actual: "d"},
		{Kind: model.DiffBodyField, Path: "body.user.name", Expected: "x", Actual: "y"},
	}

	result := rs.Apply(diffs)
	if !result[0].Ignored {
		t.Error("expected body.user.id to be ignored by wildcard rule")
	}
	if !result[1].Ignored {
		t.Error("expected body.order.id to be ignored by wildcard rule")
	}
	if result[2].Ignored {
		t.Error("expected body.user.name to NOT be ignored")
	}
}

func TestPathPattern_Wildcard_MultiLevel(t *testing.T) {
	rs := NewRuleSet([]Rule{
		{
			Name:  "ignore_deep_ts",
			Kind:  "ignore",
			Paths: []string{"**.createdAt"},
		},
	})

	diffs := []model.Difference{
		{Kind: model.DiffBodyField, Path: "body.user.createdAt", Expected: "a", Actual: "b"},
		{Kind: model.DiffBodyField, Path: "body.order.item.createdAt", Expected: "c", Actual: "d"},
		{Kind: model.DiffBodyField, Path: "body.name", Expected: "x", Actual: "y"},
	}

	result := rs.Apply(diffs)
	if !result[0].Ignored {
		t.Error("expected body.user.createdAt to be ignored by ** wildcard")
	}
	if !result[1].Ignored {
		t.Error("expected body.order.item.createdAt to be ignored by ** wildcard")
	}
	if result[2].Ignored {
		t.Error("expected body.name to NOT be ignored")
	}
}

func TestPathPattern_ArrayWildcard(t *testing.T) {
	rs := NewRuleSet([]Rule{
		{
			Name:  "ignore_array_ids",
			Kind:  "ignore",
			Paths: []string{"body.items[*].id"},
		},
	})

	diffs := []model.Difference{
		{Kind: model.DiffBodyField, Path: "body.items[0].id", Expected: "a", Actual: "b"},
		{Kind: model.DiffBodyField, Path: "body.items[5].id", Expected: "c", Actual: "d"},
		{Kind: model.DiffBodyField, Path: "body.items[0].name", Expected: "x", Actual: "y"},
	}

	result := rs.Apply(diffs)
	if !result[0].Ignored {
		t.Error("expected body.items[0].id to be ignored by [*] wildcard")
	}
	if !result[1].Ignored {
		t.Error("expected body.items[5].id to be ignored by [*] wildcard")
	}
	if result[2].Ignored {
		t.Error("expected body.items[0].name to NOT be ignored")
	}
}

func TestFormatPath(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		parts    []string
		expected string
	}{
		{"single part", "body", []string{"name"}, "body.name"},
		{"multiple parts", "body", []string{"user", "address"}, "body.user.address"},
		{"empty prefix", "", []string{"name"}, "name"},
		{"no parts", "body", nil, "body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPath(tt.prefix, tt.parts...)
			if got != tt.expected {
				t.Errorf("FormatPath(%q, %v) = %q, want %q", tt.prefix, tt.parts, got, tt.expected)
			}
		})
	}
}
