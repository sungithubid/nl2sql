package prompt

import (
	einoPrompt "github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// --- 模板 Key 常量 ---

const (
	// KeyAgentSystem React Agent 系统提示词（仅 system message，无变量）
	KeyAgentSystem = "agent_system"

	// KeyGenerateSQL SQL 生成完整模板（system + user，GoTemplate 格式支持条件块）
	KeyGenerateSQL = "generate_sql"

	// KeyBuildAnswer 回答构建完整模板（system + user，FString 格式）
	KeyBuildAnswer = "build_answer"
)

// --- 默认模板内容 ---

const defaultAgentSystemContent = `You are an expert NL2SQL (Natural Language to SQL) agent. Your goal is to answer the user's question by querying a database.

You have the following tools available:
1. **retrieve_ddl** - Search for relevant database table schemas (DDL). Always start by understanding the database structure.
2. **retrieve_docs** - Search for business documentation to understand domain-specific terms and data meanings.
3. **retrieve_sql_examples** - Search for similar SQL query examples as references.
4. **generate_sql** - Generate a SQL query based on the question and context you've gathered.
5. **execute_sql** - Execute the generated SQL query on the database.

## Workflow Guidelines:
1. **First**, use retrieve_ddl to find relevant table structures for the user's question.
2. **Optionally**, use retrieve_docs and retrieve_sql_examples if you need more context.
3. **Then**, use generate_sql to create a SQL query, passing in the context you've gathered.
4. **Finally**, use execute_sql to run the query and get results.
5. If execute_sql fails, analyze the error, modify the SQL, and try again (up to 3 retries).
6. After getting results, provide a clear natural language answer summarizing the findings.

## Important Rules:
- Only generate SELECT queries. Never use INSERT, UPDATE, DELETE, or DDL statements.
- Always add LIMIT clauses to prevent excessive data retrieval.
- Provide your final answer in natural language, clearly addressing the user's question.
- Include the SQL query you used in your final answer for transparency.`

const defaultGenerateSQLSystemContent = `You are a SQL expert. Your task is to generate a valid SQL query based on the user's question, using the provided database schema, documentation, and similar SQL examples as context.

Rules:
1. Only output the SQL query, without any explanation or markdown formatting.
2. The SQL must be syntactically correct and executable.
3. Use the database schema (DDL) to ensure correct table names, column names, and data types.
4. Reference similar SQL examples if available to understand the query patterns.
5. If documentation is provided, use it to understand business-specific terminology and data meanings.
6. If a previous error is provided, fix the SQL to avoid the same error.
7. Do not include any DML statements (INSERT, UPDATE, DELETE) - only SELECT queries are allowed.
8. Always add reasonable LIMIT clauses to prevent returning too many rows.`

// defaultGenerateSQLUserContent 使用 GoTemplate 格式，支持条件块
const defaultGenerateSQLUserContent = `{{if .ddl_context}}=== Database Schema (DDL) ===
{{.ddl_context}}

{{end}}{{if .doc_context}}=== Related Documentation ===
{{.doc_context}}

{{end}}{{if .sql_context}}=== Similar SQL Examples ===
{{.sql_context}}

{{end}}{{if .previous_error}}=== Previous Error ===
The previously generated SQL caused an error. Please fix the SQL based on the error message:
{{.previous_error}}

{{end}}=== User Question ===
{{.question}}`

const defaultBuildAnswerSystemContent = `You are a data analyst assistant. Based on the SQL query results, provide a clear and helpful natural language answer to the user's question.`

// defaultBuildAnswerUserContent 使用 FString 格式
const defaultBuildAnswerUserContent = `User asked: {question}

Generated SQL: {sql}

Query Results (JSON):
{results_json}

Please provide a clear, concise natural language answer to the user's question based on the above query results. The answer should:
1. Directly address the user's question
2. Summarize the key findings from the data
3. Use natural, conversational language
4. If there are no results, clearly state that no data was found`

// --- 注册默认模板 ---

func init() {
	// Agent 系统提示词（仅 system message，无变量，使用 FString）
	Register(KeyAgentSystem, einoPrompt.FromMessages(
		schema.FString,
		schema.SystemMessage(defaultAgentSystemContent),
	))

	// SQL 生成模板（GoTemplate 格式，支持条件块）
	Register(KeyGenerateSQL, einoPrompt.FromMessages(
		schema.GoTemplate,
		schema.SystemMessage(defaultGenerateSQLSystemContent),
		schema.UserMessage(defaultGenerateSQLUserContent),
	))

	// 回答构建模板（FString 格式）
	Register(KeyBuildAnswer, einoPrompt.FromMessages(
		schema.FString,
		schema.SystemMessage(defaultBuildAnswerSystemContent),
		schema.UserMessage(defaultBuildAnswerUserContent),
	))
}
