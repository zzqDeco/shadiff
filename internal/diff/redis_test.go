package diff

import (
	"testing"

	"shadiff/internal/model"
)

func TestCompareRedisSideEffects_Equal(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "SET", RedisKey: "user:1", RedisArgs: []string{"user:1", "ada"}},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "SET", RedisKey: "user:1", RedisArgs: []string{"user:1", "ada"}},
	}

	diffs := CompareRedisSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs for equal Redis commands, got %d: %+v", len(diffs), diffs)
	}
}

func TestCompareRedisSideEffects_DifferentCommandCount(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "SET", RedisKey: "k1", RedisArgs: []string{"k1", "v1"}},
		{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "GET", RedisKey: "k1", RedisArgs: []string{"k1"}},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "SET", RedisKey: "k1", RedisArgs: []string{"k1", "v1"}},
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
		[]model.SideEffect{{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "GET", RedisKey: "k1", RedisArgs: []string{"k1"}}},
		[]model.SideEffect{{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "DEL", RedisKey: "k1", RedisArgs: []string{"k1"}}},
	)
	if len(diffs) != 1 || diffs[0].Path != "sideEffects.redis[0].command" {
		t.Fatalf("expected command diff, got %+v", diffs)
	}
}

func TestCompareRedisSideEffects_DifferentKey(t *testing.T) {
	diffs := CompareRedisSideEffects(
		[]model.SideEffect{{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "GET", RedisKey: "k1", RedisArgs: []string{"k1"}}},
		[]model.SideEffect{{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "GET", RedisKey: "k2", RedisArgs: []string{"k2"}}},
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
		[]model.SideEffect{{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "SET", RedisKey: "k1", RedisArgs: []string{"k1", "v1"}}},
		[]model.SideEffect{{Type: model.SideEffectDB, DBType: "redis", RedisCommand: "SET", RedisKey: "k1", RedisArgs: []string{"k1", "v2"}}},
	)
	if len(diffs) != 1 || diffs[0].Path != "sideEffects.redis[0].args" {
		t.Fatalf("expected args diff, got %+v", diffs)
	}
}

func TestCompareRedisSideEffects_FiltersNonRedis(t *testing.T) {
	original := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mysql", Query: "SELECT 1"},
		{Type: model.SideEffectHTTP},
	}
	replay := []model.SideEffect{
		{Type: model.SideEffectDB, DBType: "mysql", Query: "SELECT 1"},
		{Type: model.SideEffectHTTP},
	}

	diffs := CompareRedisSideEffects(original, replay)
	if len(diffs) != 0 {
		t.Fatalf("expected 0 diffs when no Redis effects exist, got %+v", diffs)
	}
}
