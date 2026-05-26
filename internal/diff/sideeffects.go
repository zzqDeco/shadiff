package diff

import (
	"fmt"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

// SideEffectComparer compares one semantic class of side effects.
type SideEffectComparer interface {
	HandledDBTypes() []string
	Compare(original, replay []model.SideEffect) []model.Difference
}

type sideEffectComparer struct {
	handled []string
	compare func(original, replay []model.SideEffect) []model.Difference
}

func (c sideEffectComparer) HandledDBTypes() []string {
	return append([]string(nil), c.handled...)
}

func (c sideEffectComparer) Compare(original, replay []model.SideEffect) []model.Difference {
	return c.compare(original, replay)
}

var defaultSideEffectComparers = []SideEffectComparer{
	sideEffectComparer{handled: []string{dbtype.MySQL, dbtype.Postgres}, compare: CompareDBSideEffects},
	sideEffectComparer{handled: []string{dbtype.Mongo}, compare: CompareMongoSideEffects},
	sideEffectComparer{handled: []string{dbtype.Redis}, compare: CompareRedisSideEffects},
}

// CompareSideEffects compares all registered side-effect types.
func CompareSideEffects(original, replay []model.SideEffect) []model.Difference {
	return compareSideEffectsWith(defaultSideEffectComparers, original, replay)
}

func compareSideEffectsWith(comparers []SideEffectComparer, original, replay []model.SideEffect) []model.Difference {
	var diffs []model.Difference
	handled := make(map[string]bool)
	for _, comparer := range comparers {
		diffs = append(diffs, comparer.Compare(original, replay)...)
		for _, dbType := range comparer.HandledDBTypes() {
			handled[dbType] = true
		}
	}

	if otherOriginal, otherReplay := countResidualSideEffects(original, handled), countResidualSideEffects(replay, handled); otherOriginal != otherReplay {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffDBQueryCount,
			Path:     "sideEffects",
			Expected: otherOriginal,
			Actual:   otherReplay,
			Message:  fmt.Sprintf("residual side effect count differs: %d vs %d", otherOriginal, otherReplay),
			Severity: model.SeverityError,
		})
	}

	return diffs
}

func countResidualSideEffects(effects []model.SideEffect, handledDBTypes map[string]bool) int {
	count := 0
	for _, effect := range effects {
		if effect.Type != model.SideEffectDB {
			count++
			continue
		}
		if !handledDBTypes[effect.DatabaseType()] {
			count++
		}
	}
	return count
}
