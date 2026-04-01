package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/gogf/gf/v2/frame/g"

	"nl2sql/internal/logic/nl2sql/prompt"
	"nl2sql/internal/logic/nl2sql/security"
)

// --- Tool 参数定义 ---

// RetrieveParams 检索工具的参数
type RetrieveParams struct {
	Query string `json:"query" jsonschema_description:"The search query to find relevant content from vector database"`
}

// GenerateSQLParams 生成 SQL 工具的参数
type GenerateSQLParams struct {
	Question   string `json:"question" jsonschema_description:"The natural language question to generate SQL for"`
	DDLContext string `json:"ddl_context,omitempty" jsonschema_description:"Relevant database DDL schema, obtained from retrieve_ddl tool"`
	DocContext string `json:"doc_context,omitempty" jsonschema_description:"Relevant business documentation, obtained from retrieve_docs tool"`
	SQLContext string `json:"sql_context,omitempty" jsonschema_description:"Similar SQL examples, obtained from retrieve_sql_examples tool"`
	PrevError  string `json:"prev_error,omitempty" jsonschema_description:"Previous SQL execution error message for retry/fix"`
}

// ExecuteSQLParams 执行 SQL 工具的参数
type ExecuteSQLParams struct {
	SQL string `json:"sql" jsonschema_description:"The SQL query to execute. Only SELECT statements are allowed."`
}

// BuildAgentTools 创建 Agent 所需的所有工具
func BuildAgentTools(executor Executor) ([]tool.BaseTool, error) {
	// Tool 1: 检索 DDL
	retrieveDDL, err := utils.InferTool(
		"retrieve_ddl",
		"Search and retrieve relevant database DDL schema (table structures, column definitions) from the vector database. Use this tool first to understand the database structure before generating SQL.",
		func(ctx context.Context, params *RetrieveParams) (string, error) {
			g.Log().Infof(ctx, "[Agent Tool] retrieve_ddl: query=%s", params.Query)
			result, err := executor.RetrieveContext(ctx, "ddl", params.Query)
			if err != nil {
				return fmt.Sprintf("Failed to retrieve DDL: %s", err.Error()), nil
			}
			if result == "" {
				return "No relevant DDL schema found.", nil
			}
			return result, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create retrieve_ddl tool: %w", err)
	}

	// Tool 2: 检索文档
	retrieveDocs, err := utils.InferTool(
		"retrieve_docs",
		"Search and retrieve relevant business documentation from the vector database. Use this tool to understand business terminology, data meanings, and domain-specific context.",
		func(ctx context.Context, params *RetrieveParams) (string, error) {
			g.Log().Infof(ctx, "[Agent Tool] retrieve_docs: query=%s", params.Query)
			result, err := executor.RetrieveContext(ctx, "doc", params.Query)
			if err != nil {
				return fmt.Sprintf("Failed to retrieve docs: %s", err.Error()), nil
			}
			if result == "" {
				return "No relevant documentation found.", nil
			}
			return result, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create retrieve_docs tool: %w", err)
	}

	// Tool 3: 检索 SQL 示例
	retrieveSQLExamples, err := utils.InferTool(
		"retrieve_sql_examples",
		"Search and retrieve similar SQL query examples from the vector database. Use this tool to find reference SQL patterns for similar questions.",
		func(ctx context.Context, params *RetrieveParams) (string, error) {
			g.Log().Infof(ctx, "[Agent Tool] retrieve_sql_examples: query=%s", params.Query)
			result, err := executor.RetrieveContext(ctx, "sql", params.Query)
			if err != nil {
				return fmt.Sprintf("Failed to retrieve SQL examples: %s", err.Error()), nil
			}
			if result == "" {
				return "No similar SQL examples found.", nil
			}
			return result, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create retrieve_sql_examples tool: %w", err)
	}

	// Tool 4: 生成 SQL
	generateSQL, err := utils.InferTool(
		"generate_sql",
		"Generate a SQL query based on the user's question and provided context (DDL schema, documentation, SQL examples). You should call retrieve tools first to gather context before calling this tool.",
		func(ctx context.Context, params *GenerateSQLParams) (string, error) {
			g.Log().Infof(ctx, "[Agent Tool] generate_sql: question=%s", params.Question)

			// 使用 eino ChatTemplate 渲染 prompt（返回 []*schema.Message）
			messages, err := prompt.Format(ctx, prompt.KeyGenerateSQL, map[string]any{
				"question":       params.Question,
				"ddl_context":    params.DDLContext,
				"doc_context":    params.DocContext,
				"sql_context":    params.SQLContext,
				"previous_error": params.PrevError,
			})
			if err != nil {
				return fmt.Sprintf("Failed to format prompt: %s", err.Error()), nil
			}

			chatModel := executor.GetChatModel()
			resp, err := chatModel.Generate(ctx, messages)
			if err != nil {
				return fmt.Sprintf("Failed to generate SQL: %s", err.Error()), nil
			}

			sql := extractSQLFromContent(resp.Content)
			g.Log().Infof(ctx, "[Agent Tool] generate_sql result: %s", sql)
			return sql, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create generate_sql tool: %w", err)
	}

	// Tool 5: 执行 SQL
	executeSQL, err := utils.InferTool(
		"execute_sql",
		"Execute a SQL query on the target business database and return the results as JSON. Only SELECT queries are allowed. If execution fails, you can modify the SQL and try again.",
		func(ctx context.Context, params *ExecuteSQLParams) (string, error) {
			g.Log().Infof(ctx, "[Agent Tool] execute_sql: sql=%s", params.SQL)

			// 使用 sqlparser 进行 SQL 安全校验
			if r := security.ValidateSQL(params.SQL); !r.Valid {
				return fmt.Sprintf("Error: SQL validation failed: %s. Please provide a valid SELECT statement.", r.Reason), nil
			}

			results, err := executor.ExecuteSQL(ctx, params.SQL)
			if err != nil {
				return fmt.Sprintf("SQL execution error: %s\nSQL: %s", err.Error(), params.SQL), nil
			}

			jsonBytes, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return fmt.Sprintf("Failed to marshal results: %s", err.Error()), nil
			}

			return string(jsonBytes), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create execute_sql tool: %w", err)
	}

	return []tool.BaseTool{
		retrieveDDL,
		retrieveDocs,
		retrieveSQLExamples,
		generateSQL,
		executeSQL,
	}, nil
}

// extractSQLFromContent 从 LLM 内容中提取 SQL（与 generate.go 中的 extractSQL 逻辑一致）
func extractSQLFromContent(content string) string {
	content = strings.TrimSpace(content)

	if idx := strings.Index(content, "```sql"); idx != -1 {
		start := idx + len("```sql")
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + len("```")
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	return content
}
