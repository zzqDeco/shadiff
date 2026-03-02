package diff

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"shadiff/internal/model"
)

// Matcher 自定义匹配器接口
type Matcher interface {
	Name() string
	Match(path string, expected, actual any) (match bool, err error)
}

// Rule 对拍规则
type Rule struct {
	Name    string   `json:"name" yaml:"name"`       // 规则名
	Kind    string   `json:"kind" yaml:"kind"`       // ignore / custom
	Paths   []string `json:"paths" yaml:"paths"`     // 匹配的 JSON 路径 (支持 * 通配)
	Pattern string   `json:"pattern" yaml:"pattern"` // 值正则匹配 (可选)
	Matcher string   `json:"matcher" yaml:"matcher"` // 自定义匹配器名 (可选)
}

// RuleSet 规则集合
type RuleSet struct {
	Rules    []Rule
	matchers map[string]Matcher
	compiled map[string]*regexp.Regexp // 预编译路径模式
}

// NewRuleSet 创建规则集
func NewRuleSet(rules []Rule, matchers ...Matcher) *RuleSet {
	rs := &RuleSet{
		Rules:    rules,
		matchers: make(map[string]Matcher),
		compiled: make(map[string]*regexp.Regexp),
	}

	for _, m := range matchers {
		rs.matchers[m.Name()] = m
	}

	// 预编译路径通配模式
	for _, r := range rules {
		for _, p := range r.Paths {
			pattern := pathToRegexp(p)
			rs.compiled[p] = regexp.MustCompile(pattern)
		}
	}

	return rs
}

// Apply 对差异应用规则，标记被忽略的差异
func (rs *RuleSet) Apply(diffs []model.Difference) []model.Difference {
	for i := range diffs {
		for _, rule := range rs.Rules {
			if rs.matchesPath(rule, diffs[i].Path) {
				switch rule.Kind {
				case "ignore":
					diffs[i].Ignored = true
					diffs[i].Rule = rule.Name
				case "custom":
					if m, ok := rs.matchers[rule.Matcher]; ok {
						match, _ := m.Match(diffs[i].Path, diffs[i].Expected, diffs[i].Actual)
						if match {
							diffs[i].Ignored = true
							diffs[i].Rule = rule.Name
						}
					}
				}
			}
		}
	}
	return diffs
}

// matchesPath 检查差异路径是否匹配规则中的任一路径模式
func (rs *RuleSet) matchesPath(rule Rule, diffPath string) bool {
	for _, p := range rule.Paths {
		if re, ok := rs.compiled[p]; ok {
			if re.MatchString(diffPath) {
				return true
			}
		}
	}
	return false
}

// pathToRegexp 将路径通配模式转为正则
// 支持: * 匹配单级, ** 匹配多级, [*] 匹配数组下标
func pathToRegexp(pattern string) string {
	// 转义正则特殊字符
	escaped := regexp.QuoteMeta(pattern)
	// 恢复通配符
	escaped = strings.ReplaceAll(escaped, `\*\*`, `.*`)
	escaped = strings.ReplaceAll(escaped, `\*`, `[^.]*`)
	escaped = strings.ReplaceAll(escaped, `\[\*\]`, `\[\d+\]`)
	return "^" + escaped + "$"
}

// --- 内置匹配器 ---

// TimestampMatcher 时间戳匹配器，忽略时间戳类字段
type TimestampMatcher struct{}

func (TimestampMatcher) Name() string { return "timestamp" }
func (TimestampMatcher) Match(path string, expected, actual any) (bool, error) {
	// 两边都是字符串且看起来像时间戳，认为匹配
	es, eOk := expected.(string)
	as, aOk := actual.(string)
	if !eOk || !aOk {
		return false, nil
	}
	return looksLikeTimestamp(es) && looksLikeTimestamp(as), nil
}

// UUIDMatcher UUID 匹配器，忽略 UUID 字段差异
type UUIDMatcher struct{}

func (UUIDMatcher) Name() string { return "uuid" }
func (UUIDMatcher) Match(path string, expected, actual any) (bool, error) {
	es, eOk := expected.(string)
	as, aOk := actual.(string)
	if !eOk || !aOk {
		return false, nil
	}
	return looksLikeUUID(es) && looksLikeUUID(as), nil
}

// NumericToleranceMatcher 数值容差匹配器
type NumericToleranceMatcher struct {
	Tolerance float64
}

func (m NumericToleranceMatcher) Name() string { return "numeric_tolerance" }
func (m NumericToleranceMatcher) Match(path string, expected, actual any) (bool, error) {
	ef, eOk := toFloat64(expected)
	af, aOk := toFloat64(actual)
	if !eOk || !aOk {
		return false, nil
	}
	diff := ef - af
	if diff < 0 {
		diff = -diff
	}
	return diff <= m.Tolerance, nil
}

// --- 辅助函数 ---

var (
	timestampRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}`)
	uuidRe      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

func looksLikeTimestamp(s string) bool {
	return timestampRe.MatchString(s)
}

func looksLikeUUID(s string) bool {
	return uuidRe.MatchString(s)
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// DefaultRules 返回默认规则集
func DefaultRules() *RuleSet {
	rules := []Rule{
		{
			Name:  "ignore_timestamps",
			Kind:  "custom",
			Paths: []string{"**.createdAt", "**.updatedAt", "**.created_at", "**.updated_at", "**.timestamp"},
			Matcher: "timestamp",
		},
		{
			Name:  "ignore_request_ids",
			Kind:  "ignore",
			Paths: []string{"headers.X-Request-Id", "headers.X-Trace-Id", "headers.Date", "headers.Server"},
		},
	}

	return NewRuleSet(rules,
		TimestampMatcher{},
		UUIDMatcher{},
		NumericToleranceMatcher{Tolerance: 0.001},
	)
}

// DefaultIgnoreHeaders 默认忽略的响应 header
func DefaultIgnoreHeaders() []string {
	return []string{
		"Date", "X-Request-Id", "X-Trace-Id",
		"Server", "Content-Length",
	}
}

// FormatDiffSummary 格式化差异摘要
func FormatDiffSummary(results []model.DiffResult) model.DiffSummary {
	summary := model.DiffSummary{
		TotalCount: len(results),
	}

	for _, r := range results {
		if r.Match {
			summary.MatchCount++
		} else {
			summary.DiffCount++
		}
		for _, d := range r.Differences {
			if d.Ignored {
				summary.IgnoreCount++
			} else if d.Severity == model.SeverityError {
				summary.ErrorCount++
			}
		}
	}

	if summary.TotalCount > 0 {
		summary.MatchRate = float64(summary.MatchCount) / float64(summary.TotalCount)
	}

	return summary
}

// FormatPath 格式化差异路径
func FormatPath(prefix string, parts ...string) string {
	result := prefix
	for _, p := range parts {
		if result == "" {
			result = p
		} else {
			result = fmt.Sprintf("%s.%s", result, p)
		}
	}
	return result
}
