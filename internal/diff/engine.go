package diff

import (
	"fmt"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"
)

// Engine is the diff engine that compares behavioral differences between recorded and replayed records
type Engine struct {
	store       *storage.FileStore
	sessionID   string
	ruleSet     *RuleSet
	jsonDiffer  *JSONDiffer
	ignoreHeaders map[string]bool
}

// EngineConfig is the diff engine configuration
type EngineConfig struct {
	SessionID     string
	Rules         []Rule
	IgnoreOrder   bool
	IgnoreHeaders []string
}

// NewEngine creates a diff engine
func NewEngine(store *storage.FileStore, cfg EngineConfig) *Engine {
	// Build the set of headers to ignore
	ignoreHeaders := make(map[string]bool)
	for _, h := range DefaultIgnoreHeaders() {
		ignoreHeaders[h] = true
	}
	for _, h := range cfg.IgnoreHeaders {
		ignoreHeaders[h] = true
	}

	// Merge default rules with custom rules
	allRules := cfg.Rules
	ruleSet := NewRuleSet(allRules,
		TimestampMatcher{},
		UUIDMatcher{},
		NumericToleranceMatcher{Tolerance: 0.001},
	)

	return &Engine{
		store:         store,
		sessionID:     cfg.SessionID,
		ruleSet:       ruleSet,
		jsonDiffer:    &JSONDiffer{IgnoreOrder: cfg.IgnoreOrder},
		ignoreHeaders: ignoreHeaders,
	}
}

// Run executes the diff comparison and returns a list of diff results
func (e *Engine) Run() ([]model.DiffResult, error) {
	// Load recorded records
	originals, err := e.store.ListRecords(e.sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load recorded records: %w", err)
	}

	// Load replay records
	replays, err := e.store.ListReplayRecords(e.sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load replay records: %w", err)
	}

	if len(originals) == 0 {
		return nil, fmt.Errorf("session %s has no recorded records", e.sessionID)
	}
	if len(replays) == 0 {
		return nil, fmt.Errorf("session %s has no replay records, please run replay first", e.sessionID)
	}

	// Build replay record index by sequence number
	replayMap := make(map[int]model.Record, len(replays))
	for _, r := range replays {
		replayMap[r.Sequence] = r
	}

	logger.DiffEvent("diff_started",
		"session", e.sessionID,
		"original_count", len(originals),
		"replay_count", len(replays),
	)

	// Compare records one by one
	var results []model.DiffResult
	for _, orig := range originals {
		replay, exists := replayMap[orig.Sequence]
		if !exists {
			results = append(results, model.DiffResult{
				RecordID: orig.ID,
				Sequence: orig.Sequence,
				Request:  orig.Request,
				Match:    false,
				Differences: []model.Difference{{
					Kind:     model.DiffBody,
					Path:     "",
					Message:  "replay record missing",
					Severity: model.SeverityError,
				}},
			})
			continue
		}

		result := e.compareRecords(orig, replay)
		results = append(results, result)
	}

	// Save results
	if err := e.store.SaveResults(e.sessionID, results); err != nil {
		logger.Error("save diff results failed", err)
	}

	logger.DiffEvent("diff_completed",
		"session", e.sessionID,
		"total", len(results),
	)

	return results, nil
}

// compareRecords compares a pair of recorded/replayed records
func (e *Engine) compareRecords(original, replay model.Record) model.DiffResult {
	var diffs []model.Difference

	// 1. Compare status codes
	if original.Response.StatusCode != replay.Response.StatusCode {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffStatusCode,
			Path:     "statusCode",
			Expected: original.Response.StatusCode,
			Actual:   replay.Response.StatusCode,
			Message:  fmt.Sprintf("status code differs: %d vs %d", original.Response.StatusCode, replay.Response.StatusCode),
			Severity: model.SeverityError,
		})
	}

	// 2. Compare response headers
	diffs = append(diffs, e.compareHeaders(original.Response.Headers, replay.Response.Headers)...)

	// 3. Compare response body (JSON structured diff)
	if len(original.Response.Body) > 0 || len(replay.Response.Body) > 0 {
		bodyDiffs := e.jsonDiffer.Compare(original.Response.Body, replay.Response.Body)
		diffs = append(diffs, bodyDiffs...)
	}

	// 4. Compare side effects (DB operation count)
	if len(original.SideEffects) != len(replay.SideEffects) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffDBQueryCount,
			Path:     "sideEffects",
			Expected: len(original.SideEffects),
			Actual:   len(replay.SideEffects),
			Message:  fmt.Sprintf("side effect count differs: %d vs %d", len(original.SideEffects), len(replay.SideEffects)),
			Severity: model.SeverityError,
		})
	}

	// Apply rules
	diffs = e.ruleSet.Apply(diffs)

	// Determine match (ignore differences marked by rules)
	match := true
	for _, d := range diffs {
		if !d.Ignored {
			match = false
			break
		}
	}

	return model.DiffResult{
		RecordID:    original.ID,
		Sequence:    original.Sequence,
		Request:     original.Request,
		Match:       match,
		Differences: diffs,
	}
}

// compareHeaders compares response headers
func (e *Engine) compareHeaders(expected, actual map[string][]string) []model.Difference {
	var diffs []model.Difference

	for k, ev := range expected {
		if e.ignoreHeaders[k] {
			continue
		}
		av, exists := actual[k]
		if !exists {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffHeader,
				Path:     fmt.Sprintf("headers.%s", k),
				Expected: ev,
				Actual:   nil,
				Message:  fmt.Sprintf("response header missing: %s", k),
				Severity: model.SeverityWarning,
			})
		} else if fmt.Sprintf("%v", ev) != fmt.Sprintf("%v", av) {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffHeader,
				Path:     fmt.Sprintf("headers.%s", k),
				Expected: ev,
				Actual:   av,
				Message:  fmt.Sprintf("response header differs: %s", k),
				Severity: model.SeverityWarning,
			})
		}
	}

	return diffs
}
