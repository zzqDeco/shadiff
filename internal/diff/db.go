package diff

import (
	"fmt"
	"strings"

	"shadiff/internal/model"
)

// CompareDBSideEffects 比较 SQL 类数据库 (MySQL/PostgreSQL) 副作用
func CompareDBSideEffects(original, replay []model.SideEffect) []model.Difference {
	origSQL := filterByType(original, "mysql", "postgres")
	replaySQL := filterByType(replay, "mysql", "postgres")

	var diffs []model.Difference

	// 比较查询数量
	if len(origSQL) != len(replaySQL) {
		diffs = append(diffs, model.Difference{
			Kind:     model.DiffDBQueryCount,
			Path:     "sideEffects.db",
			Expected: len(origSQL),
			Actual:   len(replaySQL),
			Message:  fmt.Sprintf("SQL 查询数量不同: %d vs %d", len(origSQL), len(replaySQL)),
			Severity: model.SeverityError,
		})
	}

	// 逐条比较 SQL (按顺序配对)
	minLen := len(origSQL)
	if len(replaySQL) < minLen {
		minLen = len(replaySQL)
	}

	for i := 0; i < minLen; i++ {
		path := fmt.Sprintf("sideEffects.db[%d]", i)

		// 标准化 SQL 再比较
		origQuery := normalizeSQL(origSQL[i].Query)
		replayQuery := normalizeSQL(replaySQL[i].Query)

		if origQuery != replayQuery {
			diffs = append(diffs, model.Difference{
				Kind:     model.DiffDBQuery,
				Path:     path + ".query",
				Expected: origSQL[i].Query,
				Actual:   replaySQL[i].Query,
				Message:  "SQL 语句不同",
				Severity: model.SeverityError,
			})
		}
	}

	return diffs
}

// normalizeSQL 标准化 SQL 语句，便于比较
func normalizeSQL(sql string) string {
	// 去除多余空白
	sql = strings.TrimSpace(sql)
	sql = strings.Join(strings.Fields(sql), " ")
	// 统一大小写 (SQL 关键字)
	sql = strings.ToUpper(sql)
	return sql
}

// filterByType 过滤指定 DB 类型的副作用
func filterByType(effects []model.SideEffect, dbTypes ...string) []model.SideEffect {
	typeSet := make(map[string]bool)
	for _, t := range dbTypes {
		typeSet[t] = true
	}

	var result []model.SideEffect
	for _, e := range effects {
		if e.Type == model.SideEffectDB && typeSet[e.DBType] {
			result = append(result, e)
		}
	}
	return result
}
