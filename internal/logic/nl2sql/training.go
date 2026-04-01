package nl2sql

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"nl2sql/internal/logic/nl2sql/component"
)

// trainDDL 将 DDL schema 向量化并存入 Qdrant（内部方法）
func (s *sNl2sql) trainDDL(ctx context.Context, ddl string) (string, error) {
	collection := component.CollectionName(ctx, "ddl")
	docID := uuid.New().String()

	ids, err := s.comp.VectorStore.Store(ctx, collection, []*component.Document{
		{
			ID:      docID,
			Content: ddl,
			MetaData: map[string]any{
				"type": "ddl",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to store ddl: %w", err)
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no id returned after storing ddl")
	}
	return ids[0], nil
}

// trainDoc 将文档向量化并存入 Qdrant（内部方法）
func (s *sNl2sql) trainDoc(ctx context.Context, documentation string) (string, error) {
	collection := component.CollectionName(ctx, "doc")
	docID := uuid.New().String()

	ids, err := s.comp.VectorStore.Store(ctx, collection, []*component.Document{
		{
			ID:      docID,
			Content: documentation,
			MetaData: map[string]any{
				"type": "doc",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to store documentation: %w", err)
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no id returned after storing documentation")
	}
	return ids[0], nil
}

// trainSQL 将问题-SQL对向量化并存入 Qdrant（内部方法）
func (s *sNl2sql) trainSQL(ctx context.Context, question, sql string) (string, error) {
	collection := component.CollectionName(ctx, "sql")
	docID := uuid.New().String()

	// 将问题和SQL拼接存储，问题作为主内容用于向量检索，SQL放在metadata中
	content := fmt.Sprintf("Question: %s\nSQL: %s", question, sql)
	ids, err := s.comp.VectorStore.Store(ctx, collection, []*component.Document{
		{
			ID:      docID,
			Content: content,
			MetaData: map[string]any{
				"type":     "sql",
				"question": question,
				"sql":      sql,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to store sql example: %w", err)
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no id returned after storing sql example")
	}
	return ids[0], nil
}
