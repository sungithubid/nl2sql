package nl2sql

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/sync/errgroup"

	"nl2sql/internal/logic/nl2sql/component"
	"nl2sql/internal/logic/nl2sql/prompt"
)

// GenerateSQL 使用 RAG 模式生成 SQL（实现 workflow.Executor 接口）
// 1. 对 question 进行一次 embedding
// 2. 并发查询 3 个 qdrant collection（ddl, doc, sql）
// 3. 构建 prompt
// 4. 调用 ChatModel 生成 SQL
func (s *sNl2sql) GenerateSQL(ctx context.Context, question string, previousError string) (string, error) {
	// Step 1: 只 embedding 一次
	vector, err := s.comp.VectorStore.EmbedQuery(ctx, question)
	if err != nil {
		return "", fmt.Errorf("failed to embed question: %w", err)
	}

	// Step 2: 并发查询 3 个 collection
	var ddlContext, docContext, sqlContext string

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		collection := component.CollectionName(egCtx, "ddl")
		result, err := s.comp.VectorStore.RetrieveContentByVector(egCtx, collection, vector, 5)
		if err != nil {
			g.Log().Warningf(egCtx, "retrieve ddl context failed: %v", err)
			return nil // 不中断其他查询
		}
		ddlContext = result
		return nil
	})
	eg.Go(func() error {
		collection := component.CollectionName(egCtx, "doc")
		result, err := s.comp.VectorStore.RetrieveContentByVector(egCtx, collection, vector, 5)
		if err != nil {
			g.Log().Warningf(egCtx, "retrieve doc context failed: %v", err)
			return nil
		}
		docContext = result
		return nil
	})
	eg.Go(func() error {
		collection := component.CollectionName(egCtx, "sql")
		result, err := s.comp.VectorStore.RetrieveContentByVector(egCtx, collection, vector, 5)
		if err != nil {
			g.Log().Warningf(egCtx, "retrieve sql context failed: %v", err)
			return nil
		}
		sqlContext = result
		return nil
	})
	if err := eg.Wait(); err != nil {
		return "", fmt.Errorf("concurrent retrieval failed: %w", err)
	}

	// Step 3: 使用 eino ChatTemplate 渲染 prompt（返回 []*schema.Message）
	messages, err := prompt.Format(ctx, prompt.KeyGenerateSQL, map[string]any{
		"question":       question,
		"ddl_context":    ddlContext,
		"doc_context":    docContext,
		"sql_context":    sqlContext,
		"previous_error": previousError,
	})
	if err != nil {
		return "", fmt.Errorf("failed to format generate_sql prompt: %w", err)
	}

	// Step 4: 调用 ChatModel 生成 SQL
	resp, err := s.comp.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate SQL: %w", err)
	}

	sql := extractSQL(resp.Content)
	return sql, nil
}

// RetrieveContext 从指定类型的 collection 中检索相关上下文（实现 workflow.Executor 接口）
// 供 Agent 工具等场景单独调用（这些场景中 query 可能不同，无法预共享 vector）
func (s *sNl2sql) RetrieveContext(ctx context.Context, docType string, query string) (string, error) {
	collection := component.CollectionName(ctx, docType)
	return s.comp.VectorStore.RetrieveContent(ctx, collection, query, 5)
}

// extractSQL 从 LLM 响应中提取纯 SQL
func extractSQL(content string) string {
	content = strings.TrimSpace(content)

	// 尝试提取 ```sql ... ``` 代码块
	if idx := strings.Index(content, "```sql"); idx != -1 {
		start := idx + len("```sql")
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// 尝试提取 ``` ... ``` 代码块
	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + len("```")
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	return content
}
