package diff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"shadiff/internal/config"

	"gopkg.in/yaml.v3"
)

// RulesFromConfig converts config rules into diff engine rules.
func RulesFromConfig(rules []config.Rule) []Rule {
	converted := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		converted = append(converted, Rule{
			Name:    rule.Name,
			Kind:    rule.Kind,
			Paths:   append([]string(nil), rule.Paths...),
			Pattern: rule.Pattern,
			Matcher: rule.Matcher,
		})
	}
	return converted
}

// LoadRulesFile loads custom rules from a JSON or YAML file.
func LoadRulesFile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rules []Rule
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &rules); err != nil {
			return nil, fmt.Errorf("parse yaml rules: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &rules); err != nil {
			return nil, fmt.Errorf("parse json rules: %w", err)
		}
	}

	return rules, nil
}
