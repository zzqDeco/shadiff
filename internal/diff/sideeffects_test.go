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
