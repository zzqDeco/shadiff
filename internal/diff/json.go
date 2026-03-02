package diff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"shadiff/internal/model"
)

// JSONDiffer is a structured JSON difference comparator
type JSONDiffer struct {
	IgnoreOrder bool // ignore array order
}

// Compare compares two JSON bodies and returns a list of differences
func (d *JSONDiffer) Compare(expected, actual []byte) []model.Difference {
	var ev, av any

	if err := json.Unmarshal(expected, &ev); err != nil {
		// expected is not valid JSON, fall back to byte comparison
		if string(expected) != string(actual) {
			return []model.Difference{{
				Kind:     model.DiffBody,
				Path:     "body",
				Expected: string(expected),
				Actual:   string(actual),
				Message:  "response body mismatch (non-JSON)",
				Severity: model.SeverityError,
			}}
		}
		return nil
	}

	if err := json.Unmarshal(actual, &av); err != nil {
		return []model.Difference{{
			Kind:     model.DiffBody,
			Path:     "body",
			Expected: string(expected),
			Actual:   string(actual),
			Message:  "recorded response is JSON but replay response is not",
			Severity: model.SeverityError,
		}}
	}

	return d.compareValues("body", ev, av)
}

// compareValues recursively compares two values
func (d *JSONDiffer) compareValues(path string, expected, actual any) []model.Difference {
	// types differ
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		// special handling: json.Unmarshal may mix int and float
		ef, eOk := toFloat64(expected)
		af, aOk := toFloat64(actual)
		if eOk && aOk {
			if ef != af {
				return []model.Difference{{
					Kind:     model.DiffBodyField,
					Path:     path,
					Expected: expected,
					Actual:   actual,
					Message:  fmt.Sprintf("numeric value differs: %v != %v", expected, actual),
					Severity: model.SeverityError,
				}}
			}
			return nil
		}

		return []model.Difference{{
			Kind:     model.DiffBodyField,
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Message:  fmt.Sprintf("type differs: %T vs %T", expected, actual),
			Severity: model.SeverityError,
		}}
	}

	switch ev := expected.(type) {
	case map[string]any:
		return d.compareObjects(path, ev, actual.(map[string]any))
	case []any:
		return d.compareArrays(path, ev, actual.([]any))
	default:
		if !reflect.DeepEqual(expected, actual) {
			return []model.Difference{{
				Kind:     model.DiffBodyField,
				Path:     path,
				Expected: expected,
				Actual:   actual,
				Message:  fmt.Sprintf("%v != %v", expected, actual),
				Severity: model.SeverityError,
			}}
		}
		return nil
	}
}

// compareObjects compares two JSON objects
func (d *JSONDiffer) compareObjects(path string, expected, actual map[string]any) []model.Difference {
	var diffs []model.Difference

	// collect all keys
	allKeys := make(map[string]bool)
	for k := range expected {
		allKeys[k] = true
	}
	for k := range actual {
		allKeys[k] = true
	}

	// sort keys for stable output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fieldPath := FormatPath(path, k)
		ev, eExists := expected[k]
		av, aExists := actual[k]

		if eExists && !aExists {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffBodyField,
				Path:     fieldPath,
				Expected: ev,
				Actual:   nil,
				Message:  "field missing",
				Severity: model.SeverityError,
			})
		} else if !eExists && aExists {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffBodyField,
				Path:     fieldPath,
				Expected: nil,
				Actual:   av,
				Message:  "extra field",
				Severity: model.SeverityWarning,
			})
		} else {
			diffs = append(diffs, d.compareValues(fieldPath, ev, av)...)
		}
	}

	return diffs
}

// compareArrays compares two JSON arrays
func (d *JSONDiffer) compareArrays(path string, expected, actual []any) []model.Difference {
	var diffs []model.Difference

	if len(expected) != len(actual) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffBodyField,
			Path:     path,
			Expected: len(expected),
			Actual:   len(actual),
			Message:  fmt.Sprintf("array length differs: %d vs %d", len(expected), len(actual)),
			Severity: model.SeverityError,
		})
	}

	if d.IgnoreOrder {
		diffs = append(diffs, d.compareArraysUnordered(path, expected, actual)...)
	} else {
		minLen := len(expected)
		if len(actual) < minLen {
			minLen = len(actual)
		}
		for i := 0; i < minLen; i++ {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			diffs = append(diffs, d.compareValues(itemPath, expected[i], actual[i])...)
		}
	}

	return diffs
}

// compareArraysUnordered compares arrays without order (attempts best-match pairing)
func (d *JSONDiffer) compareArraysUnordered(path string, expected, actual []any) []model.Difference {
	var diffs []model.Difference
	used := make([]bool, len(actual))

	for i, ev := range expected {
		found := false
		for j, av := range actual {
			if used[j] {
				continue
			}
			subDiffs := d.compareValues(fmt.Sprintf("%s[%d]", path, i), ev, av)
			if len(subDiffs) == 0 {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffBodyField,
				Path:     fmt.Sprintf("%s[%d]", path, i),
				Expected: ev,
				Actual:   nil,
				Message:  "array element not found in replay result",
				Severity: model.SeverityError,
			})
		}
	}

	for j, av := range actual {
		if !used[j] {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffBodyField,
				Path:     fmt.Sprintf("%s[extra]", path),
				Expected: nil,
				Actual:   av,
				Message:  "extra array element in replay result",
				Severity: model.SeverityWarning,
			})
		}
	}

	return diffs
}
