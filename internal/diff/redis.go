package diff

import (
	"fmt"
	"slices"

	"shadiff/internal/dbtype"
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
		orig := origRedis[i].Redis()
		rep := replayRedis[i].Redis()

		if orig.Command != rep.Command {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffRedisCommand,
				Path:     path + ".command",
				Expected: orig.Command,
				Actual:   rep.Command,
				Message:  "Redis command differs",
				Severity: model.SeverityError,
			})
		}
		if orig.Key != rep.Key {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffRedisCommand,
				Path:     path + ".key",
				Expected: orig.Key,
				Actual:   rep.Key,
				Message:  "Redis key differs",
				Severity: model.SeverityError,
			})
		}
		if !slices.Equal(orig.Args, rep.Args) {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffRedisCommand,
				Path:     path + ".args",
				Expected: orig.Args,
				Actual:   rep.Args,
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
		if e.Type == model.SideEffectDB && e.DatabaseType() == dbtype.Redis && e.Redis() != nil {
			result = append(result, e)
		}
	}
	return result
}
