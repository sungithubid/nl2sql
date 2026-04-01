package workflow

import (
	"context"
	"fmt"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"

	"nl2sql/internal/logic/nl2sql/prompt"
	"nl2sql/internal/model"
)

// CompileAgent 创建 React Agent
// Agent 自主决策工具调用顺序: retrieve_ddl → retrieve_docs → retrieve_sql_examples → generate_sql → execute_sql
// 注意: 需要传入独立的 ChatModel 实例，因为 react.NewAgent 会通过 BindTools 修改 ChatModel 内部状态，
// 如果共享同一个实例会导致 simple/retry 模式的请求也携带 tools 参数。
func CompileAgent(ctx context.Context, executor Executor, agentChatModel einoModel.ChatModel) (*react.Agent, error) {
	// 创建 Agent 工具
	tools, err := BuildAgentTools(executor)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent tools: %w", err)
	}

	// 使用独立的 ChatModel（用作 Agent 的大脑，避免 BindTools 污染共享实例）
	chatModel := agentChatModel

	// 创建 React Agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		// 注入系统提示词（从 eino ChatTemplate 获取）
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			systemMsgs, err := prompt.Format(ctx, prompt.KeyAgentSystem, nil)
			if err != nil {
				g.Log().Warningf(ctx, "failed to format agent system prompt: %v", err)
				return input
			}
			return append(systemMsgs, input...)
		},
		MaxStep: 25, // 允许多轮工具调用（检索+生成+执行+重试）
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create react agent: %w", err)
	}

	g.Log().Info(ctx, "React Agent compiled successfully")
	return agent, nil
}

// RunAgent 使用 React Agent 执行 NL2SQL
func RunAgent(ctx context.Context, agent *react.Agent, question string) (*model.AskOutput, error) {
	// 构建用户消息
	input := []*schema.Message{
		schema.UserMessage(question),
	}

	// 调用 Agent
	resp, err := agent.Generate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	g.Log().Infof(ctx, "Agent final response: %s", resp.Content)

	return &model.AskOutput{
		Question: question,
		Answer:   resp.Content,
	}, nil
}
