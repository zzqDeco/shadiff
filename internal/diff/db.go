package diff

import (
	"fmt"
	"strings"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

// CompareDBSideEffects compares SQL database (MySQL/PostgreSQL) side effects
func CompareDBSideEffects(original, replay []model.SideEffect) []model.Difference {
	origSQL := filterByType(original, dbtype.MySQL, dbtype.Postgres)
	replaySQL := filterByType(replay, dbtype.MySQL, dbtype.Postgres)

	var diffs []model.Difference

	// Compare query count
	if len(origSQL) != len(replaySQL) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffDBQueryCount,
			Path:     "sideEffects.db",
			Expected: len(origSQL),
			Actual:   len(replaySQL),
			Message:  fmt.Sprintf("SQL query count differs: %d vs %d", len(origSQL), len(replaySQL)),
			Severity: model.SeverityError,
		})
	}

	// Compare SQL statements one by one (paired by order)
	minLen := len(origSQL)
	if len(replaySQL) < minLen {
		minLen = len(replaySQL)
	}

	for i := 0; i < minLen; i++ {
		path := fmt.Sprintf("sideEffects.db[%d]", i)
		origPayload := origSQL[i].SQL()
		replayPayload := replaySQL[i].SQL()

		// Normalize SQL before comparing
		origQuery := normalizeSQL(origPayload.Query)
		replayQuery := normalizeSQL(replayPayload.Query)

		if origQuery != replayQuery {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffDBQuery,
				Path:     path + ".query",
				Expected: origPayload.Query,
				Actual:   replayPayload.Query,
				Message:  "SQL statement differs",
				Severity: model.SeverityError,
			})
		}
	}

	return diffs
}

// normalizeSQL normalizes a SQL statement for comparison
func normalizeSQL(sql string) string {
	// Remove extra whitespace
	sql = strings.TrimSpace(sql)
	sql = strings.Join(strings.Fields(sql), " ")
	// Normalize case (SQL keywords)
	sql = strings.ToUpper(sql)
	return sql
}

// filterByType filters side effects by the specified DB types
func filterByType(effects []model.SideEffect, dbTypes ...string) []model.SideEffect {
	typeSet := make(map[string]bool)
	for _, t := range dbTypes {
		typeSet[t] = true
	}

	var result []model.SideEffect
	for _, e := range effects {
		if e.Type == model.SideEffectDB && e.SQL() != nil && typeSet[e.DatabaseType()] {
			result = append(result, e)
		}
	}
	return result
}
