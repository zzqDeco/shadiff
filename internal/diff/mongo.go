package diff

import (
	"fmt"

	"shadiff/internal/model"
)

// CompareMongoSideEffects compares MongoDB side effects
func CompareMongoSideEffects(original, replay []model.SideEffect) []model.Difference {
	origMongo := filterMongoEffects(original)
	replayMongo := filterMongoEffects(replay)

	var diffs []model.Difference

	// Compare operation count
	if len(origMongo) != len(replayMongo) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffMongoOp,
			Path:     "sideEffects.mongo",
			Expected: len(origMongo),
			Actual:   len(replayMongo),
			Message:  fmt.Sprintf("MongoDB operation count differs: %d vs %d", len(origMongo), len(replayMongo)),
			Severity: model.SeverityError,
		})
	}

	// Compare one by one (paired by order)
	minLen := len(origMongo)
	if len(replayMongo) < minLen {
		minLen = len(replayMongo)
	}

	for i := 0; i < minLen; i++ {
		path := fmt.Sprintf("sideEffects.mongo[%d]", i)
		orig := origMongo[i]
		rep := replayMongo[i]

		// Compare collection name
		if orig.Collection != rep.Collection {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffMongoOp,
				Path:     path + ".collection",
				Expected: orig.Collection,
				Actual:   rep.Collection,
				Message:  "MongoDB collection differs",
				Severity: model.SeverityError,
			})
		}

		// Compare operation type
		if orig.Operation != rep.Operation {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffMongoOp,
				Path:     path + ".operation",
				Expected: orig.Operation,
				Actual:   rep.Operation,
				Message:  "MongoDB operation type differs",
				Severity: model.SeverityError,
			})
		}

		// Compare database name
		if orig.Database != rep.Database {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffMongoOp,
				Path:     path + ".database",
				Expected: orig.Database,
				Actual:   rep.Database,
				Message:  "MongoDB database differs",
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
