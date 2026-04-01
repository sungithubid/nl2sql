package nl2sql

import (
	"context"
	"fmt"

	"nl2sql/internal/logic/nl2sql/security"
)

// ExecuteSQL 在目标数据库上执行 SQL，返回结果集（实现 workflow.Executor 接口）
// 数据库连接池在 Init 阶段创建单例，通过 s.comp.TargetDB 复用
func (s *sNl2sql) ExecuteSQL(ctx context.Context, sql string) ([]map[string]any, error) {
	if s.comp == nil || s.comp.TargetDB == nil {
		return nil, fmt.Errorf("NL2SQL service not initialized: target database not available")
	}

	// 安全兜底：在数据库执行前校验 SQL
	if r := security.ValidateSQL(sql); !r.Valid {
		return nil, fmt.Errorf("SQL rejected by security validation: %s", r.Reason)
	}

	result, err := s.comp.TargetDB.GetAll(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("SQL execution error: %w", err)
	}

	rows := make([]map[string]any, 0, len(result))
	for _, record := range result {
		row := make(map[string]any)
		for k, v := range record {
			row[k] = v.Val()
		}
		rows = append(rows, row)
	}

	return rows, nil
}
