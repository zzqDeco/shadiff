package diff

import (
	"testing"

	"shadiff/internal/model"
)

func TestCompareMongoSideEffects_Equal(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "testdb"},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "testdb"},
	}

	diffs := CompareMongoSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for equal mongo ops, got %d: %v", len(diffs), diffs)
	}
}

func TestCompareMongoSideEffects_DifferentOperationCount(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "testdb"},
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "orders", Operation: "insert", Database: "testdb"},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "testdb"},
	}

	diffs := CompareMongoSideEffects(original, replay)
	found := false
	for _, d := range diffs {
		if d.Kind == model.DiffMongoOp && d.Path == "sideEffects.mongo" {
			found = true
			if d.Expected != 2 || d.Actual != 1 {
				t.Errorf("expected count 2 vs 1, got %v vs %v", d.Expected, d.Actual)
			}
			break
		}
	}
	if !found {
		t.Error("expected a mongo operation count diff")
	}
}

func TestCompareMongoSideEffects_DifferentCollection(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "testdb"},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "orders", Operation: "find", Database: "testdb"},
	}

	diffs := CompareMongoSideEffects(original, replay)
	found := false
	for _, d := range diffs {
		if d.Message == "MongoDB collection differs" {
			found = true
			if d.Expected != "users" || d.Actual != "orders" {
				t.Errorf("expected users vs orders, got %v vs %v", d.Expected, d.Actual)
			}
			if d.Severity != model.SeverityError {
				t.Errorf("expected severity error, got %s", d.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected a collection diff")
	}
}

func TestCompareMongoSideEffects_DifferentOperation(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "testdb"},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "insert", Database: "testdb"},
	}

	diffs := CompareMongoSideEffects(original, replay)
	found := false
	for _, d := range diffs {
		if d.Message == "MongoDB operation type differs" {
			found = true
			if d.Expected != "find" || d.Actual != "insert" {
				t.Errorf("expected find vs insert, got %v vs %v", d.Expected, d.Actual)
			}
			break
		}
	}
	if !found {
		t.Error("expected an operation type diff")
	}
}

func TestCompareMongoSideEffects_DifferentDatabase(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "db1"},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mongo", Collection: "users", Operation: "find", Database: "db2"},
	}

	diffs := CompareMongoSideEffects(original, replay)
	found := false
	for _, d := range diffs {
		if d.Message == "MongoDB database differs" {
			found = true
			if d.Expected != "db1" || d.Actual != "db2" {
				t.Errorf("expected db1 vs db2, got %v vs %v", d.Expected, d.Actual)
			}
			if d.Severity != model.SeverityWarning {
				t.Errorf("expected severity warning for database diff, got %s", d.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected a database name diff")
	}
}

func TestCompareMongoSideEffects_FiltersNonMongo(t *testing.T) {
	// Non-mongo side effects should be ignored
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mysql", Query: "SELECT 1"},
		{Type: model.SideEffectHTTP},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mysql", Query: "SELECT 1"},
		{Type: model.SideEffectHTTP},
	}

	diffs := CompareMongoSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs when no mongo effects exist, got %d: %v", len(diffs), diffs)
	}
}

func TestCompareMongoSideEffects_EmptyLists(t *testing.T) {
	diffs := CompareMongoSideEffects(nil, nil)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for empty lists, got %d", len(diffs))
	}
}
