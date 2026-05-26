package diff

import (
	"testing"

	"shadiff/internal/model"
)

func testSQLSideEffect(dbType, query string) model.SideEffect {
	return model.NewSQLSideEffect(dbType, query, 0)
}

func TestCompareDBSideEffects_Equal(t *testing.T) {
	original := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT * FROM users"),
		testSQLSideEffect("postgres", "INSERT INTO orders (id) VALUES (1)"),
	}
	replay := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT * FROM users"),
		testSQLSideEffect("postgres", "INSERT INTO orders (id) VALUES (1)"),
	}

	diffs := CompareDBSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for equal SQL ops, got %d: %v", len(diffs), diffs)
	}
}

func TestCompareDBSideEffects_DifferentQueryCount(t *testing.T) {
	original := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT 1"),
		testSQLSideEffect("mysql", "SELECT 2"),
	}
	replay := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT 1"),
	}

	diffs := CompareDBSideEffects(original, replay)
	found := false
	for _, d := range diffs {
		if d.Kind == model.DiffDBQueryCount {
			found = true
			if d.Expected != 2 || d.Actual != 1 {
				t.Errorf("expected count 2 vs 1, got %v vs %v", d.Expected, d.Actual)
			}
			break
		}
	}
	if !found {
		t.Error("expected a DiffDBQueryCount diff")
	}
}

func TestCompareDBSideEffects_DifferentSQL(t *testing.T) {
	original := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT * FROM users"),
	}
	replay := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT * FROM orders"),
	}

	diffs := CompareDBSideEffects(original, replay)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Kind != model.DiffDBQuery {
		t.Errorf("expected kind %s, got %s", model.DiffDBQuery, diffs[0].Kind)
	}
	if diffs[0].Message != "SQL statement differs" {
		t.Errorf("unexpected message: %s", diffs[0].Message)
	}
}

func TestCompareDBSideEffects_SQLNormalization(t *testing.T) {
	// Extra whitespace and different casing should be normalized away
	original := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT  *  FROM   users"),
	}
	replay := []model.SideEffect{
		testSQLSideEffect("mysql", "select * from users"),
	}

	diffs := CompareDBSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs after SQL normalization, got %d: %v", len(diffs), diffs)
	}
}

func TestCompareDBSideEffects_MixedDBTypes(t *testing.T) {
	// Only mysql and postgres should be filtered; mongo should be ignored
	original := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT 1"),
		model.NewMongoSideEffect(model.MongoSideEffect{Collection: "users", Operation: "find"}, 0),
		{Type: model.SideEffectHTTP},
	}
	replay := []model.SideEffect{
		testSQLSideEffect("mysql", "SELECT 1"),
		model.NewMongoSideEffect(model.MongoSideEffect{Collection: "users", Operation: "find"}, 0),
		{Type: model.SideEffectHTTP},
	}

	diffs := CompareDBSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs (mongo and HTTP filtered out), got %d: %v", len(diffs), diffs)
	}
}

func TestCompareDBSideEffects_EmptyLists(t *testing.T) {
	diffs := CompareDBSideEffects(nil, nil)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for empty lists, got %d", len(diffs))
	}
}

func TestNormalizeSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"trims whitespace", "  SELECT 1  ", "SELECT 1"},
		{"collapses spaces", "SELECT  *  FROM   users", "SELECT * FROM USERS"},
		{"uppercases", "select * from users", "SELECT * FROM USERS"},
		{"handles tabs and newlines", "SELECT\n\t*\n\tFROM users", "SELECT * FROM USERS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSQL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeSQL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
