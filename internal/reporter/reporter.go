package reporter

import (
	"fmt"
	"io"

	"shadiff/internal/model"
)

// Reporter is the report generator interface
type Reporter interface {
	Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error
}

// NewReporter creates a report generator based on the specified format
func NewReporter(format string) (Reporter, error) {
	switch format {
	case "terminal", "":
		return &TerminalReporter{}, nil
	case "json":
		return &JSONReporter{}, nil
	case "html":
		return &HTMLReporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported report format: %s", format)
	}
}
