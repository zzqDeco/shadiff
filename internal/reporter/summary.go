package reporter

import "shadiff/internal/model"

// DifferenceSummary groups differences into operator-facing categories.
type DifferenceSummary struct {
	HTTP              int `json:"http"`
	SQL               int `json:"sql"`
	MongoDB           int `json:"mongoDb"`
	Redis             int `json:"redis"`
	UnknownSideEffect int `json:"unknownSideEffect"`
	Ignored           int `json:"ignored"`
}

// Total returns the number of categorized non-ignored differences.
func (s DifferenceSummary) Total() int {
	return s.HTTP + s.SQL + s.MongoDB + s.Redis + s.UnknownSideEffect
}

// SummarizeDifferences counts non-ignored differences by report category.
func SummarizeDifferences(results []model.DiffResult) DifferenceSummary {
	var summary DifferenceSummary
	for _, result := range results {
		for _, diff := range result.Differences {
			if diff.Ignored {
				summary.Ignored++
				continue
			}
			switch diff.Kind {
			case model.DiffStatusCode, model.DiffHeader, model.DiffBody, model.DiffBodyField:
				summary.HTTP++
			case model.DiffDBQuery:
				summary.SQL++
			case model.DiffDBQueryCount:
				if diff.Path == "sideEffects.db" {
					summary.SQL++
				} else {
					summary.UnknownSideEffect++
				}
			case model.DiffMongoOp:
				summary.MongoDB++
			case model.DiffRedisCommand, model.DiffRedisCount:
				summary.Redis++
			case model.DiffExternalCall:
				summary.UnknownSideEffect++
			default:
				summary.UnknownSideEffect++
			}
		}
	}
	return summary
}
