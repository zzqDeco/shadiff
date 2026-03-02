package diff

import (
	"fmt"

	"shadiff/internal/logger"
	"shadiff/internal/model"
	"shadiff/internal/storage"
)

// Engine 对拍引擎，比较录制和回放记录的行为差异
type Engine struct {
	store       *storage.FileStore
	sessionID   string
	ruleSet     *RuleSet
	jsonDiffer  *JSONDiffer
	ignoreHeaders map[string]bool
}

// EngineConfig 对拍引擎配置
type EngineConfig struct {
	SessionID     string
	Rules         []Rule
	IgnoreOrder   bool
	IgnoreHeaders []string
}

// NewEngine 创建对拍引擎
func NewEngine(store *storage.FileStore, cfg EngineConfig) *Engine {
	// 构建忽略 header 集合
	ignoreHeaders := make(map[string]bool)
	for _, h := range DefaultIgnoreHeaders() {
		ignoreHeaders[h] = true
	}
	for _, h := range cfg.IgnoreHeaders {
		ignoreHeaders[h] = true
	}

	// 合并默认规则和自定义规则
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

// Run 执行对拍，返回差异结果列表
func (e *Engine) Run() ([]model.DiffResult, error) {
	// 读取录制记录
	originals, err := e.store.ListRecords(e.sessionID)
	if err != nil {
		return nil, fmt.Errorf("读取录制记录失败: %w", err)
	}

	// 读取回放记录
	replays, err := e.store.ListReplayRecords(e.sessionID)
	if err != nil {
		return nil, fmt.Errorf("读取回放记录失败: %w", err)
	}

	if len(originals) == 0 {
		return nil, fmt.Errorf("会话 %s 没有录制记录", e.sessionID)
	}
	if len(replays) == 0 {
		return nil, fmt.Errorf("会话 %s 没有回放记录，请先执行 replay", e.sessionID)
	}

	// 按序号建立回放记录索引
	replayMap := make(map[int]model.Record, len(replays))
	for _, r := range replays {
		replayMap[r.Sequence] = r
	}

	logger.DiffEvent("diff_started",
		"session", e.sessionID,
		"original_count", len(originals),
		"replay_count", len(replays),
	)

	// 逐条对拍
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
					Message:  "回放记录缺失",
					Severity: model.SeverityError,
				}},
			})
			continue
		}

		result := e.compareRecords(orig, replay)
		results = append(results, result)
	}

	// 保存结果
	if err := e.store.SaveResults(e.sessionID, results); err != nil {
		logger.Error("save diff results failed", err)
	}

	logger.DiffEvent("diff_completed",
		"session", e.sessionID,
		"total", len(results),
	)

	return results, nil
}

// compareRecords 比较一对录制/回放记录
func (e *Engine) compareRecords(original, replay model.Record) model.DiffResult {
	var diffs []model.Difference

	// 1. 比较状态码
	if original.Response.StatusCode != replay.Response.StatusCode {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffStatusCode,
			Path:     "statusCode",
			Expected: original.Response.StatusCode,
			Actual:   replay.Response.StatusCode,
			Message:  fmt.Sprintf("状态码不同: %d vs %d", original.Response.StatusCode, replay.Response.StatusCode),
			Severity: model.SeverityError,
		})
	}

	// 2. 比较响应 headers
	diffs = append(diffs, e.compareHeaders(original.Response.Headers, replay.Response.Headers)...)

	// 3. 比较响应 body (JSON 结构化对比)
	if len(original.Response.Body) > 0 || len(replay.Response.Body) > 0 {
		bodyDiffs := e.jsonDiffer.Compare(original.Response.Body, replay.Response.Body)
		diffs = append(diffs, bodyDiffs...)
	}

	// 4. 比较副作用 (DB 操作数量)
	if len(original.SideEffects) != len(replay.SideEffects) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffDBQueryCount,
			Path:     "sideEffects",
			Expected: len(original.SideEffects),
			Actual:   len(replay.SideEffects),
			Message:  fmt.Sprintf("副作用数量不同: %d vs %d", len(original.SideEffects), len(replay.SideEffects)),
			Severity: model.SeverityError,
		})
	}

	// 应用规则
	diffs = e.ruleSet.Apply(diffs)

	// 判断是否匹配（忽略被规则标记的差异）
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

// compareHeaders 比较响应 headers
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
				Message:  fmt.Sprintf("响应 header 缺失: %s", k),
				Severity: model.SeverityWarning,
			})
		} else if fmt.Sprintf("%v", ev) != fmt.Sprintf("%v", av) {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffHeader,
				Path:     fmt.Sprintf("headers.%s", k),
				Expected: ev,
				Actual:   av,
				Message:  fmt.Sprintf("响应 header 不同: %s", k),
				Severity: model.SeverityWarning,
			})
		}
	}

	return diffs
}
