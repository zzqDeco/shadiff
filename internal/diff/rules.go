package diff

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"shadiff/internal/model"
)

// Matcher is a custom matcher interface
type Matcher interface {
	Name() string
	Match(path string, expected, actual any) (match bool, err error)
}

// Rule is a diff rule
type Rule struct {
	Name    string   `json:"name" yaml:"name"`       // rule name
	Kind    string   `json:"kind" yaml:"kind"`       // ignore / custom
	Paths   []string `json:"paths" yaml:"paths"`     // JSON paths to match (supports * wildcard)
	Pattern string   `json:"pattern" yaml:"pattern"` // value regex match (optional)
	Matcher string   `json:"matcher" yaml:"matcher"` // custom matcher name (optional)
}

// RuleSet is a collection of rules
type RuleSet struct {
	Rules    []Rule
	matchers map[string]Matcher
	compiled map[string]*regexp.Regexp // pre-compiled path patterns
}

// NewRuleSet creates a rule set
func NewRuleSet(rules []Rule, matchers ...Matcher) *RuleSet {
	rs := &RuleSet{
		Rules:    rules,
		matchers: make(map[string]Matcher),
		compiled: make(map[string]*regexp.Regexp),
	}

	for _, m := range matchers {
		rs.matchers[m.Name()] = m
	}

	// Pre-compile path wildcard patterns
	for _, r := range rules {
		for _, p := range r.Paths {
			pattern := pathToRegexp(p)
			rs.compiled[p] = regexp.MustCompile(pattern)
		}
	}

	return rs
}

// Apply applies rules to differences and marks ignored differences
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

// matchesPath checks whether the diff path matches any path pattern in the rule
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

// pathToRegexp converts a path wildcard pattern to a regex
// Supports: * matches single level, ** matches multiple levels, [*] matches array index
func pathToRegexp(pattern string) string {
	// Escape regex special characters
	escaped := regexp.QuoteMeta(pattern)
	// Restore wildcards
	escaped = strings.ReplaceAll(escaped, `\*\*`, `.*`)
	escaped = strings.ReplaceAll(escaped, `\*`, `[^.]*`)
	escaped = strings.ReplaceAll(escaped, `\[\*\]`, `\[\d+\]`)
	return "^" + escaped + "$"
}

// --- Built-in matchers ---

// TimestampMatcher is a timestamp matcher that ignores timestamp-like fields
type TimestampMatcher struct{}

func (TimestampMatcher) Name() string { return "timestamp" }
func (TimestampMatcher) Match(path string, expected, actual any) (bool, error) {
	// If both sides are strings and look like timestamps, consider them a match
	es, eOk := expected.(string)
	as, aOk := actual.(string)
	if !eOk || !aOk {
		return false, nil
	}
	return looksLikeTimestamp(es) && looksLikeTimestamp(as), nil
}

// UUIDMatcher is a UUID matcher that ignores UUID field differences
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

// NumericToleranceMatcher is a numeric tolerance matcher
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

// --- Helper functions ---

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

// DefaultRules returns the default rule set
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

// DefaultIgnoreHeaders returns the default response headers to ignore
func DefaultIgnoreHeaders() []string {
	return []string{
		"Date", "X-Request-Id", "X-Trace-Id",
		"Server", "Content-Length",
	}
}

// FormatDiffSummary formats a diff summary
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

// FormatPath formats a diff path
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
