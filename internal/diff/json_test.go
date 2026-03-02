package diff

import (
	"testing"

	"shadiff/internal/model"
)

func TestJSONDiffer_IdenticalObjects(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"name":"alice","age":30}`)
	actual := []byte(`{"name":"alice","age":30}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical objects, got %d: %v", len(diffs), diffs)
	}
}

func TestJSONDiffer_IdenticalArrays(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`[1,2,3]`)
	actual := []byte(`[1,2,3]`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical arrays, got %d: %v", len(diffs), diffs)
	}
}

func TestJSONDiffer_DifferentStringValues(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"name":"alice"}`)
	actual := []byte(`{"name":"bob"}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Kind != model.DiffBodyField {
		t.Errorf("expected kind %s, got %s", model.DiffBodyField, diffs[0].Kind)
	}
	if diffs[0].Path != "body.name" {
		t.Errorf("expected path body.name, got %s", diffs[0].Path)
	}
}

func TestJSONDiffer_DifferentNumberValues(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"count":10}`)
	actual := []byte(`{"count":20}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != "body.count" {
		t.Errorf("expected path body.count, got %s", diffs[0].Path)
	}
}

func TestJSONDiffer_DifferentBoolValues(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"active":true}`)
	actual := []byte(`{"active":false}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != "body.active" {
		t.Errorf("expected path body.active, got %s", diffs[0].Path)
	}
}

func TestJSONDiffer_MissingField(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"name":"alice","age":30}`)
	actual := []byte(`{"name":"alice"}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for missing field, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Message != "field missing" {
		t.Errorf("expected message 'field missing', got %q", diffs[0].Message)
	}
	if diffs[0].Path != "body.age" {
		t.Errorf("expected path body.age, got %s", diffs[0].Path)
	}
	if diffs[0].Severity != model.SeverityError {
		t.Errorf("expected severity error, got %s", diffs[0].Severity)
	}
}

func TestJSONDiffer_ExtraField(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"name":"alice"}`)
	actual := []byte(`{"name":"alice","extra":"value"}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for extra field, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Message != "extra field" {
		t.Errorf("expected message 'extra field', got %q", diffs[0].Message)
	}
	if diffs[0].Severity != model.SeverityWarning {
		t.Errorf("expected severity warning for extra field, got %s", diffs[0].Severity)
	}
}

func TestJSONDiffer_ArrayLengthMismatch(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"items":[1,2,3]}`)
	actual := []byte(`{"items":[1,2]}`)

	diffs := d.Compare(expected, actual)
	// Should include length diff + no element diff since elements match within minLen
	found := false
	for _, diff := range diffs {
		if diff.Message == "array length differs: 3 vs 2" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected array length mismatch diff, got: %v", diffs)
	}
}

func TestJSONDiffer_NestedObjectDifferences(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"user":{"address":{"city":"NYC"}}}`)
	actual := []byte(`{"user":{"address":{"city":"LA"}}}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != "body.user.address.city" {
		t.Errorf("expected path body.user.address.city, got %s", diffs[0].Path)
	}
}

func TestJSONDiffer_IgnoreOrder_MatchingElements(t *testing.T) {
	d := &JSONDiffer{IgnoreOrder: true}
	expected := []byte(`[1,2,3]`)
	actual := []byte(`[3,1,2]`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs with IgnoreOrder for same elements, got %d: %v", len(diffs), diffs)
	}
}

func TestJSONDiffer_IgnoreOrder_MissingElement(t *testing.T) {
	d := &JSONDiffer{IgnoreOrder: true}
	expected := []byte(`[1,2,3]`)
	actual := []byte(`[1,2,4]`)

	diffs := d.Compare(expected, actual)
	if len(diffs) == 0 {
		t.Error("expected diffs when array element missing with IgnoreOrder")
	}
}

func TestJSONDiffer_NullVsValue(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"name":null}`)
	actual := []byte(`{"name":"alice"}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for null vs value, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Path != "body.name" {
		t.Errorf("expected path body.name, got %s", diffs[0].Path)
	}
}

func TestJSONDiffer_TypeMismatch_StringVsNumber(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"value":"hello"}`)
	actual := []byte(`{"value":42}`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for type mismatch, got %d: %v", len(diffs), diffs)
	}
	if diffs[0].Kind != model.DiffBodyField {
		t.Errorf("expected kind %s, got %s", model.DiffBodyField, diffs[0].Kind)
	}
	// Should report type mismatch
	if diffs[0].Message == "" {
		t.Error("expected non-empty message for type mismatch")
	}
}

func TestJSONDiffer_NonJSONBodyMismatch(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`plain text body`)
	actual := []byte(`different body`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff for non-JSON mismatch, got %d", len(diffs))
	}
	if diffs[0].Kind != model.DiffBody {
		t.Errorf("expected kind %s, got %s", model.DiffBody, diffs[0].Kind)
	}
}

func TestJSONDiffer_NonJSONBodyIdentical(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`plain text body`)
	actual := []byte(`plain text body`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for identical non-JSON bodies, got %d", len(diffs))
	}
}

func TestJSONDiffer_ExpectedJSONActualNot(t *testing.T) {
	d := &JSONDiffer{}
	expected := []byte(`{"valid":"json"}`)
	actual := []byte(`not json at all`)

	diffs := d.Compare(expected, actual)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Message != "recorded response is JSON but replay response is not" {
		t.Errorf("unexpected message: %s", diffs[0].Message)
	}
}
