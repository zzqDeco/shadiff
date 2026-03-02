package diff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"shadiff/internal/model"
)

// JSONDiffer JSON 结构化差异比较器
type JSONDiffer struct {
	IgnoreOrder bool // 忽略数组顺序
}

// Compare 比较两个 JSON body，返回差异列表
func (d *JSONDiffer) Compare(expected, actual []byte) []model.Difference {
	var ev, av any

	if err := json.Unmarshal(expected, &ev); err != nil {
		// expected 不是有效 JSON，做字节比较
		if string(expected) != string(actual) {
			return []model.Difference{{
				Kind:     model.DiffBody,
				Path:     "body",
				Expected: string(expected),
				Actual:   string(actual),
				Message:  "响应 body 不一致 (非 JSON)",
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
			Message:  "录制为 JSON 但回放非 JSON",
			Severity: model.SeverityError,
		}}
	}

	return d.compareValues("body", ev, av)
}

// compareValues 递归比较两个值
func (d *JSONDiffer) compareValues(path string, expected, actual any) []model.Difference {
	// 类型不同
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		// 特殊处理：json.Unmarshal 可能将 int 和 float 混合
		ef, eOk := toFloat64(expected)
		af, aOk := toFloat64(actual)
		if eOk && aOk {
			if ef != af {
				return []model.Difference{{
					Kind:     model.DiffBodyField,
					Path:     path,
					Expected: expected,
					Actual:   actual,
					Message:  fmt.Sprintf("数值不同: %v != %v", expected, actual),
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
			Message:  fmt.Sprintf("类型不同: %T vs %T", expected, actual),
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

// compareObjects 比较两个 JSON 对象
func (d *JSONDiffer) compareObjects(path string, expected, actual map[string]any) []model.Difference {
	var diffs []model.Difference

	// 收集所有 key
	allKeys := make(map[string]bool)
	for k := range expected {
		allKeys[k] = true
	}
	for k := range actual {
		allKeys[k] = true
	}

	// 排序 key 以保证输出稳定
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
				Message:  "字段缺失",
				Severity: model.SeverityError,
			})
		} else if !eExists && aExists {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffBodyField,
				Path:     fieldPath,
				Expected: nil,
				Actual:   av,
				Message:  "多出字段",
				Severity: model.SeverityWarning,
			})
		} else {
			diffs = append(diffs, d.compareValues(fieldPath, ev, av)...)
		}
	}

	return diffs
}

// compareArrays 比较两个 JSON 数组
func (d *JSONDiffer) compareArrays(path string, expected, actual []any) []model.Difference {
	var diffs []model.Difference

	if len(expected) != len(actual) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffBodyField,
			Path:     path,
			Expected: len(expected),
			Actual:   len(actual),
			Message:  fmt.Sprintf("数组长度不同: %d vs %d", len(expected), len(actual)),
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

// compareArraysUnordered 无序比较数组（尝试匹配最佳配对）
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
				Message:  "数组元素在回放结果中未找到匹配",
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
				Message:  "回放结果中多出的数组元素",
				Severity: model.SeverityWarning,
			})
		}
	}

	return diffs
}
