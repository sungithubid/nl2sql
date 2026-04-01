package nl2sql

import (
	"context"
	"encoding/json"
	"fmt"

	"nl2sql/internal/logic/nl2sql/prompt"
)

// BuildAnswer 使用 ChatModel 将查询结果转化为自然语言回答（实现 workflow.Executor 接口）
func (s *sNl2sql) BuildAnswer(ctx context.Context, question string, sql string, results []map[string]interface{}) (string, error) {
	resultsJSON, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	messages, err := prompt.Format(ctx, prompt.KeyBuildAnswer, map[string]any{
		"question":     question,
		"sql":          sql,
		"results_json": string(resultsJSON),
	})
	if err != nil {
		return "", fmt.Errorf("failed to format build_answer prompt: %w", err)
	}

	resp, err := s.comp.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to build answer: %w", err)
	}

	return resp.Content, nil
}
