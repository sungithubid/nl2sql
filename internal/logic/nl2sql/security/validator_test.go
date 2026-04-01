package security

import (
	"strings"
	"testing"
)

func TestValidateSQL(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantValid bool
		wantInMsg string // 校验失败时，Reason 中应包含此子串
	}{
		// --- 合法 SELECT ---
		{name: "simple SELECT", sql: "SELECT * FROM users", wantValid: true},
		{name: "SELECT with WHERE", sql: "SELECT id, name FROM users WHERE age > 18", wantValid: true},
		{name: "SELECT with JOIN", sql: "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id", wantValid: true},
		{name: "SELECT with subquery", sql: "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE amount > 100)", wantValid: true},
		{name: "SELECT with UNION", sql: "SELECT name FROM customers UNION SELECT name FROM suppliers", wantValid: true},
		{name: "SELECT with GROUP BY", sql: "SELECT department, COUNT(*) FROM employees GROUP BY department HAVING COUNT(*) > 5", wantValid: true},
		{name: "SELECT with LIMIT", sql: "SELECT * FROM users LIMIT 10", wantValid: true},
		{name: "SELECT with ORDER BY", sql: "SELECT * FROM users ORDER BY created_at DESC LIMIT 20", wantValid: true},
		{name: "SELECT with aggregate", sql: "SELECT MAX(salary), MIN(salary), AVG(salary) FROM employees", wantValid: true},
		{name: "SELECT with aliases", sql: "SELECT u.name AS user_name FROM users AS u WHERE u.id = 1", wantValid: true},

		// --- 拒绝空 SQL ---
		{name: "empty SQL", sql: "", wantValid: false, wantInMsg: "empty"},
		{name: "whitespace only", sql: "   ", wantValid: false, wantInMsg: "empty"},

		// --- 拒绝非 SELECT ---
		{name: "INSERT", sql: "INSERT INTO users (name) VALUES ('Alice')", wantValid: false, wantInMsg: "only SELECT"},
		{name: "UPDATE", sql: "UPDATE users SET age = 31 WHERE name = 'Alice'", wantValid: false, wantInMsg: "only SELECT"},
		{name: "DELETE", sql: "DELETE FROM users WHERE id = 1", wantValid: false, wantInMsg: "only SELECT"},
		{name: "DROP TABLE", sql: "DROP TABLE users", wantValid: false, wantInMsg: "only SELECT"},
		{name: "CREATE TABLE", sql: "CREATE TABLE test (id INT PRIMARY KEY)", wantValid: false, wantInMsg: "only SELECT"},
		{name: "ALTER TABLE", sql: "ALTER TABLE users ADD COLUMN email VARCHAR(255)", wantValid: false, wantInMsg: "only SELECT"},
		{name: "TRUNCATE", sql: "TRUNCATE TABLE users", wantValid: false, wantInMsg: "only SELECT"},

		// --- 拒绝多语句 ---
		{name: "multi: SELECT then DROP", sql: "SELECT 1; DROP TABLE users", wantValid: false, wantInMsg: "multi-statement"},
		{name: "multi: SELECT then INSERT", sql: "SELECT * FROM users; INSERT INTO logs VALUES (1, 'hack')", wantValid: false, wantInMsg: "multi-statement"},

		// --- 拒绝 INTO OUTFILE ---
		{name: "INTO OUTFILE", sql: "SELECT * FROM users INTO OUTFILE '/tmp/dump.txt'", wantValid: false, wantInMsg: "OUTFILE"},
		{name: "INTO DUMPFILE", sql: "SELECT * FROM users INTO DUMPFILE '/tmp/dump.txt'", wantValid: false, wantInMsg: "DUMPFILE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSQL(tt.sql)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateSQL(%q) valid = %v, want %v, reason: %s", tt.sql, result.Valid, tt.wantValid, result.Reason)
			}
			if !result.Valid && tt.wantInMsg != "" {
				if !strings.Contains(result.Reason, tt.wantInMsg) {
					t.Errorf("ValidateSQL(%q) reason = %q, want to contain %q", tt.sql, result.Reason, tt.wantInMsg)
				}
			}
		})
	}
}
