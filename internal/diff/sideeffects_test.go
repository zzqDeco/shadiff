package diff

import (
	"reflect"
	"testing"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

func TestDefaultSideEffectComparersCoverSupportedDBTypes(t *testing.T) {
	got := make(map[string]bool)
	for _, comparer := range defaultSideEffectComparers {
		for _, dbType := range comparer.HandledDBTypes() {
			got[dbType] = true
		}
	}

	want := make(map[string]bool)
	for _, dbType := range dbtype.Supported() {
		want[dbType] = true
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default comparers cover %#v, want %#v", got, want)
	}
}

func TestCompareSideEffects_UnknownDBTypeCountsAsResidual(t *testing.T) {
	original := []model.SideEffect{{
		Type: model.SideEffectDB,
		Database: &model.DatabaseSideEffect{
			Type: "sqlite",
			SQL:  &model.SQLSideEffect{Query: "SELECT 1"},
		},
	}}

	diffs := CompareSideEffects(original, nil)
	if len(diffs) != 1 {
		t.Fatalf("diff count = %d, want 1: %+v", len(diffs), diffs)
	}
	if diffs[0].Path != "sideEffects" || diffs[0].Expected != 1 || diffs[0].Actual != 0 {
		t.Fatalf("unexpected residual diff: %+v", diffs[0])
	}
}

func TestCompareSideEffects_RegistryDispatchesSQLMongoAndRedis(t *testing.T) {
	original := []model.SideEffect{
		model.NewSQLSideEffect(dbtype.MySQL, "SELECT * FROM users", 0),
		model.NewMongoSideEffect(model.MongoSideEffect{
			Database:   "app",
			Collection: "users",
			Operation:  "find",
		}, 0),
		model.NewRedisSideEffect("SET", "user:1", []string{"user:1", "old"}, 0),
	}
	replay := []model.SideEffect{
		model.NewSQLSideEffect(dbtype.MySQL, "SELECT * FROM accounts", 0),
		model.NewMongoSideEffect(model.MongoSideEffect{
			Database:   "app",
			Collection: "orders",
			Operation:  "find",
		}, 0),
		model.NewRedisSideEffect("SET", "user:1", []string{"user:1", "new"}, 0),
	}

	diffs := CompareSideEffects(original, replay)
	for _, want := range []model.DifferenceKind{model.DiffDBQuery, model.DiffMongoOp, model.DiffRedisCommand} {
		if !hasDifferenceKind(diffs, want) {
			t.Fatalf("CompareSideEffects() missing %s diff: %+v", want, diffs)
		}
	}
}

func TestCompareSideEffects_HandledCustomDBTypeIsNotResidual(t *testing.T) {
	original := []model.SideEffect{
		{
			Type: model.SideEffectDB,
			Database: &model.DatabaseSideEffect{
				Type: "sqlite",
				SQL:  &model.SQLSideEffect{Query: "SELECT 1"},
			},
		},
		{Type: model.SideEffectHTTP},
	}
	comparers := []SideEffectComparer{
		sideEffectComparer{
			handled: []string{"sqlite"},
			compare: func(original, replay []model.SideEffect) []model.Difference {
				return nil
			},
		},
	}

	diffs := compareSideEffectsWith(comparers, original, nil)
	if len(diffs) != 1 {
		t.Fatalf("diff count = %d, want residual HTTP-only diff: %+v", len(diffs), diffs)
	}
	if diffs[0].Path != "sideEffects" || diffs[0].Expected != 1 || diffs[0].Actual != 0 {
		t.Fatalf("unexpected residual diff: %+v", diffs[0])
	}
}

func hasDifferenceKind(diffs []model.Difference, kind model.DifferenceKind) bool {
	for _, diff := range diffs {
		if diff.Kind == kind {
			return true
		}
	}
	return false
}
