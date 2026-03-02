package diff

import (
	"fmt"

	"shadiff/internal/model"
)

// CompareMongoSideEffects 比较 MongoDB 副作用
func CompareMongoSideEffects(original, replay []model.SideEffect) []model.Difference {
	origMongo := filterMongoEffects(original)
	replayMongo := filterMongoEffects(replay)

	var diffs []model.Difference

	// 比较操作数量
	if len(origMongo) != len(replayMongo) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffMongoOp,
			Path:     "sideEffects.mongo",
			Expected: len(origMongo),
			Actual:   len(replayMongo),
			Message:  fmt.Sprintf("MongoDB 操作数量不同: %d vs %d", len(origMongo), len(replayMongo)),
			Severity: model.SeverityError,
		})
	}

	// 逐条比较 (按顺序配对)
	minLen := len(origMongo)
	if len(replayMongo) < minLen {
		minLen = len(replayMongo)
	}

	for i := 0; i < minLen; i++ {
		path := fmt.Sprintf("sideEffects.mongo[%d]", i)
		orig := origMongo[i]
		rep := replayMongo[i]

		// 比较集合名
		if orig.Collection != rep.Collection {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffMongoOp,
				Path:     path + ".collection",
				Expected: orig.Collection,
				Actual:   rep.Collection,
				Message:  "MongoDB 集合不同",
				Severity: model.SeverityError,
			})
		}

		// 比较操作类型
		if orig.Operation != rep.Operation {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffMongoOp,
				Path:     path + ".operation",
				Expected: orig.Operation,
				Actual:   rep.Operation,
				Message:  "MongoDB 操作类型不同",
				Severity: model.SeverityError,
			})
		}

		// 比较数据库名
		if orig.Database != rep.Database {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffMongoOp,
				Path:     path + ".database",
				Expected: orig.Database,
				Actual:   rep.Database,
				Message:  "MongoDB 数据库不同",
				Severity: model.SeverityWarning,
			})
		}
	}

	return diffs
}

func filterMongoEffects(effects []model.SideEffect) []model.SideEffect {
	var result []model.SideEffect
	for _, e := range effects {
		if e.Type == model.SideEffectDB && e.DBType == "mongo" {
			result = append(result, e)
		}
	}
	return result
}
