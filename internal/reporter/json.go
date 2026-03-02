package reporter

import (
	"encoding/json"
	"io"

	"shadiff/internal/model"
)

// JSONReporter JSON 格式报告
type JSONReporter struct{}

// jsonReport JSON 报告结构
type jsonReport struct {
	Summary model.DiffSummary  `json:"summary"`
	Results []model.DiffResult `json:"results"`
}

func (r *JSONReporter) Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error {
	report := jsonReport{
		Summary: summary,
		Results: results,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
