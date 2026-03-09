package diff

import (
	"os"
	"path/filepath"
	"testing"

	"shadiff/internal/config"
	"shadiff/internal/model"
)

func TestRulesFromConfig(t *testing.T) {
	rules := RulesFromConfig([]config.Rule{
		{
			Name:    "ignore_id",
			Kind:    "ignore",
			Paths:   []string{"body.id"},
			Pattern: ".*",
			Matcher: "timestamp",
		},
	})

	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
	if rules[0].Name != "ignore_id" || rules[0].Matcher != "timestamp" {
		t.Fatalf("unexpected converted rule: %+v", rules[0])
	}
}

func TestLoadRulesFile_JSONAndYAML(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "rules.json")
	yamlPath := filepath.Join(dir, "rules.yaml")

	if err := os.WriteFile(jsonPath, []byte(`[{"name":"json-rule","kind":"ignore","paths":["body.id"]}]`), 0644); err != nil {
		t.Fatalf("write json rules: %v", err)
	}
	if err := os.WriteFile(yamlPath, []byte("- name: yaml-rule\n  kind: ignore\n  paths:\n    - body.createdAt\n"), 0644); err != nil {
		t.Fatalf("write yaml rules: %v", err)
	}

	jsonRules, err := LoadRulesFile(jsonPath)
	if err != nil {
		t.Fatalf("LoadRulesFile(json) error: %v", err)
	}
	if len(jsonRules) != 1 || jsonRules[0].Name != "json-rule" {
		t.Fatalf("unexpected json rules: %+v", jsonRules)
	}

	yamlRules, err := LoadRulesFile(yamlPath)
	if err != nil {
		t.Fatalf("LoadRulesFile(yaml) error: %v", err)
	}
	if len(yamlRules) != 1 || yamlRules[0].Name != "yaml-rule" {
		t.Fatalf("unexpected yaml rules: %+v", yamlRules)
	}
}

func TestEngine_LimitDiffs(t *testing.T) {
	engine := &Engine{maxDiffs: 1}
	diffs := []model.Difference{
		{Message: "first", Severity: model.SeverityError},
		{Message: "second", Severity: model.SeverityError},
	}

	limited := engine.limitDiffs(diffs, false)
	if len(limited) != 2 {
		t.Fatalf("len(limited) = %d, want 2", len(limited))
	}
	if limited[1].Message != "differences truncated after 1 items" {
		t.Fatalf("unexpected synthetic diff: %+v", limited[1])
	}
}
