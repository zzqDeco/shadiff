package reporter

import (
	"encoding/json"
	"io"

	"shadiff/internal/model"
)

// JSONReporter generates reports in JSON format
type JSONReporter struct{}

// jsonReport is the JSON report structure
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
