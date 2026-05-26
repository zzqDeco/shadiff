package diff

import (
	"fmt"
	"slices"

	"shadiff/internal/model"
)

// CompareRedisSideEffects compares Redis command side effects.
func CompareRedisSideEffects(original, replay []model.SideEffect) []model.Difference {
	origRedis := filterRedisEffects(original)
	replayRedis := filterRedisEffects(replay)

	var diffs []model.Difference
	if len(origRedis) != len(replayRedis) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffRedisCount,
			Path:     "sideEffects.redis",
			Expected: len(origRedis),
			Actual:   len(replayRedis),
			Message:  fmt.Sprintf("Redis command count differs: %d vs %d", len(origRedis), len(replayRedis)),
			Severity: model.SeverityError,
		})
	}

	minLen := len(origRedis)
	if len(replayRedis) < minLen {
		minLen = len(replayRedis)
	}

	for i := 0; i < minLen; i++ {
		path := fmt.Sprintf("sideEffects.redis[%d]", i)
		orig := origRedis[i]
		rep := replayRedis[i]

		if orig.RedisCommand != rep.RedisCommand {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffRedisCommand,
				Path:     path + ".command",
				Expected: orig.RedisCommand,
				Actual:   rep.RedisCommand,
				Message:  "Redis command differs",
				Severity: model.SeverityError,
			})
		}
		if orig.RedisKey != rep.RedisKey {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffRedisCommand,
				Path:     path + ".key",
				Expected: orig.RedisKey,
				Actual:   rep.RedisKey,
				Message:  "Redis key differs",
				Severity: model.SeverityError,
			})
		}
		if !slices.Equal(orig.RedisArgs, rep.RedisArgs) {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffRedisCommand,
				Path:     path + ".args",
				Expected: orig.RedisArgs,
				Actual:   rep.RedisArgs,
				Message:  "Redis command arguments differ",
				Severity: model.SeverityError,
			})
		}
	}

	return diffs
}

func filterRedisEffects(effects []model.SideEffect) []model.SideEffect {
	var result []model.SideEffect
	for _, e := range effects {
		if e.Type == model.SideEffectDB && e.DBType == "redis" {
			result = append(result, e)
		}
	}
	return result
}
