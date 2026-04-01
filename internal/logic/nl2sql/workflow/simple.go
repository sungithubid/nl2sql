package workflow

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"

	internalModel "nl2sql/internal/model"
)

// Executor 定义工作流各步骤的执行接口
type Executor interface {
	GenerateSQL(ctx context.Context, question string, previousError string) (string, error)
	ExecuteSQL(ctx context.Context, sql string) ([]map[string]interface{}, error)
	BuildAnswer(ctx context.Context, question string, sql string, results []map[string]interface{}) (string, error)
	RetrieveContext(ctx context.Context, docType string, query string) (string, error)
	GetChatModel() model.ChatModel
}

// CompileSimpleChain 使用 Eino Chain 编排简单工作流: generate_sql → execute_sql → build_answer
// 返回编译后的 Runnable，可多次调用 Invoke 执行
func CompileSimpleChain(ctx context.Context, executor Executor) (compose.Runnable[*WorkflowState, *WorkflowState], error) {
	chain := compose.NewChain[*WorkflowState, *WorkflowState]()

	// Node 1: 生成 SQL
	generateSQLNode := compose.InvokableLambda(func(ctx context.Context, state *WorkflowState) (*WorkflowState, error) {
		sql, err := executor.GenerateSQL(ctx, state.Question, state.PreviousError)
		if err != nil {
			return nil, fmt.Errorf("generate SQL failed: %w", err)
		}
		state.SQL = sql
		return state, nil
	})

	// Node 2: 执行 SQL（启用 callback 以支持 SQL 安全校验）
	executeSQLNode := compose.InvokableLambda(func(ctx context.Context, state *WorkflowState) (*WorkflowState, error) {
		// 检查 callback 校验结果
		if err := checkSQLValidation(ctx); err != nil {
			return nil, err
		}
		results, err := executor.ExecuteSQL(ctx, state.SQL)
		if err != nil {
			return nil, fmt.Errorf("execute SQL failed: %w", err)
		}
		state.Results = results
		return state, nil
	}, compose.WithLambdaCallbackEnable(true))

	// Node 3: 构建自然语言回答
	buildAnswerNode := compose.InvokableLambda(func(ctx context.Context, state *WorkflowState) (*WorkflowState, error) {
		answer, err := executor.BuildAnswer(ctx, state.Question, state.SQL, state.Results)
		if err != nil {
			return nil, fmt.Errorf("build answer failed: %w", err)
		}
		state.Answer = answer
		return state, nil
	})

	// 组装 Chain: generateSQL → executeSQL → buildAnswer
	// 给 execute_sql 节点设置显式 key，以支持 DesignateNode 定向 callback
	chain.AppendLambda(generateSQLNode)
	chain.AppendLambda(executeSQLNode, compose.WithNodeKey(nodeKeyExecuteSQL))
	chain.AppendLambda(buildAnswerNode)

	// 编译
	return chain.Compile(ctx)
}

// RunSimple 使用编译后的 Chain 执行简单工作流
func RunSimple(ctx context.Context, runnable compose.Runnable[*WorkflowState, *WorkflowState], question string) (*internalModel.AskOutput, error) {
	state := &WorkflowState{
		Question: question,
	}

	// 传入 SQL 安全校验 callback，仅作用于 execute_sql 节点
	handler := newSQLValidationHandler()
	result, err := runnable.Invoke(ctx, state, compose.WithCallbacks(handler).DesignateNode(nodeKeyExecuteSQL))
	if err != nil {
		return nil, err
	}

	return &internalModel.AskOutput{
		Question: result.Question,
		SQL:      result.SQL,
		Results:  result.Results,
		Answer:   result.Answer,
	}, nil
}
