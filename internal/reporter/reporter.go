package reporter

import (
	"fmt"
	"io"

	"shadiff/internal/model"
)

// Reporter 报告生成器接口
type Reporter interface {
	Generate(results []model.DiffResult, summary model.DiffSummary, w io.Writer) error
}

// NewReporter 根据格式创建报告生成器
func NewReporter(format string) (Reporter, error) {
	switch format {
	case "terminal", "":
		return &TerminalReporter{}, nil
	case "json":
		return &JSONReporter{}, nil
	case "html":
		return &HTMLReporter{}, nil
	default:
		return nil, fmt.Errorf("不支持的报告格式: %s", format)
	}
}
