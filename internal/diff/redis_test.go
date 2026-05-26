package diff

import (
	"testing"

	"shadiff/internal/model"
)

func testRedisSideEffect(command, key string, args ...string) model.SideEffect {
	return model.NewRedisSideEffect(command, key, args, 0)
}

func TestCompareRedisSideEffects_Equal(t *testing.T) {
	original := []model.SideEffect{
		testRedisSideEffect("SET", "user:1", "user:1", "ada"),
	}
	replay := []model.SideEffect{
		testRedisSideEffect("SET", "user:1", "user:1", "ada"),
	}

	diffs := CompareRedisSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs for equal Redis commands, got %d: %+v", len(diffs), diffs)
	}
}

func TestCompareRedisSideEffects_DifferentCommandCount(t *testing.T) {
	original := []model.SideEffect{
		testRedisSideEffect("SET", "k1", "k1", "v1"),
		testRedisSideEffect("GET", "k1", "k1"),
	}
	replay := []model.SideEffect{
		testRedisSideEffect("SET", "k1", "k1", "v1"),
	}

	diffs := CompareRedisSideEffects(original, replay)
	if len(diffs) == 0 || diffs[0].Kind != model.DiffRedisCount {
		t.Fatalf("expected redis count diff, got %+v", diffs)
	}
	if diffs[0].Expected != 2 || diffs[0].Actual != 1 {
		t.Fatalf("count diff = %v vs %v, want 2 vs 1", diffs[0].Expected, diffs[0].Actual)
	}
}

func TestCompareRedisSideEffects_DifferentCommand(t *testing.T) {
	diffs := CompareRedisSideEffects(
		[]model.SideEffect{testRedisSideEffect("GET", "k1", "k1")},
		[]model.SideEffect{testRedisSideEffect("DEL", "k1", "k1")},
	)
	if len(diffs) != 1 || diffs[0].Path != "sideEffects.redis[0].command" {
		t.Fatalf("expected command diff, got %+v", diffs)
	}
}

func TestCompareRedisSideEffects_DifferentKey(t *testing.T) {
	diffs := CompareRedisSideEffects(
		[]model.SideEffect{testRedisSideEffect("GET", "k1", "k1")},
		[]model.SideEffect{testRedisSideEffect("GET", "k2", "k2")},
	)
	if len(diffs) != 2 {
		t.Fatalf("expected key and args diffs, got %+v", diffs)
	}
	if diffs[0].Path != "sideEffects.redis[0].key" || diffs[1].Path != "sideEffects.redis[0].args" {
		t.Fatalf("unexpected diffs: %+v", diffs)
	}
}

func TestCompareRedisSideEffects_DifferentArgs(t *testing.T) {
	diffs := CompareRedisSideEffects(
		[]model.SideEffect{testRedisSideEffect("SET", "k1", "k1", "v1")},
		[]model.SideEffect{testRedisSideEffect("SET", "k1", "k1", "v2")},
	)
	if len(diffs) != 1 || diffs[0].Path != "sideEffects.redis[0].args" {
		t.Fatalf("expected args diff, got %+v", diffs)
	}
}

func TestCompareRedisSideEffects_FiltersNonRedis(t *testing.T) {
	original := []model.SideEffect{
		model.NewSQLSideEffect("mysql", "SELECT 1", 0),
		{Type: model.SideEffectHTTP},
	}
	replay := []model.SideEffect{
		model.NewSQLSideEffect("mysql", "SELECT 1", 0),
		{Type: model.SideEffectHTTP},
	}

	diffs := CompareRedisSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs when no Redis effects exist, got %+v", diffs)
	}
}
