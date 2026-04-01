package security

import (
	"fmt"
	"strings"

	"github.com/xwb1989/sqlparser"
)

// SQLValidationResult SQL 校验结果
type SQLValidationResult struct {
	Valid  bool
	Reason string
}

// ValidateSQL 使用 sqlparser 校验 SQL 是否为安全的 SELECT 语句
// 校验规则：
//  1. 拒绝空 SQL
//  2. 拒绝多语句（分号分隔的多条 SQL）
//  3. 拒绝 INTO OUTFILE / INTO DUMPFILE
//  4. 仅允许 SELECT 语句（含 UNION SELECT）
func ValidateSQL(sql string) SQLValidationResult {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return SQLValidationResult{Valid: false, Reason: "empty SQL statement"}
	}

	// 拒绝多语句：将 SQL 按分号拆分，只允许单条语句
	stmts, err := sqlparser.SplitStatementToPieces(sql)
	if err != nil {
		return SQLValidationResult{Valid: false, Reason: fmt.Sprintf("SQL parse error: %v", err)}
	}
	if len(stmts) != 1 {
		return SQLValidationResult{Valid: false, Reason: "multi-statement SQL is not allowed, only single SELECT is permitted"}
	}

	// 拒绝 INTO OUTFILE / INTO DUMPFILE（字符串级检查，因为 sqlparser 不支持该语法）
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "INTO OUTFILE") || strings.Contains(upper, "INTO DUMPFILE") {
		return SQLValidationResult{Valid: false, Reason: "SELECT INTO OUTFILE/DUMPFILE is not allowed"}
	}

	// 解析 SQL 语法树
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return SQLValidationResult{Valid: false, Reason: fmt.Sprintf("SQL syntax error: %v", err)}
	}

	// 仅允许 SELECT 或 UNION SELECT
	switch stmt.(type) {
	case *sqlparser.Select, *sqlparser.Union:
		return SQLValidationResult{Valid: true}
	default:
		return SQLValidationResult{Valid: false, Reason: "only SELECT statements are allowed"}
	}
}
