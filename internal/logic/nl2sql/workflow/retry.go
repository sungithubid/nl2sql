package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/gogf/gf/v2/frame/g"

	internalModel "nl2sql/internal/model"
)

const maxRetries = 3

// nodeKeyGenerateSQL Graph 节点名
const (
	nodeKeyGenerateSQL = "generate_sql"
	nodeKeyExecuteSQL  = "execute_sql"
	nodeKeyBuildAnswer = "build_answer"
	nodeKeyHandleError = "handle_error"
)

// CompileRetryGraph 使用 Eino Graph 编排重试工作流
// generate_sql → execute_sql ─┬─ success → build_answer → end
//
//	▲                  │
//	│             fail+retry
//	└──────────────────┘
//	                   │
//	              fail+max → handle_error → end
func CompileRetryGraph(ctx context.Context, executor Executor) (compose.Runnable[*WorkflowState, *WorkflowState], error) {
	graph := compose.NewGraph[*WorkflowState, *WorkflowState]()

	// Node 1: 生成 SQL
	generateSQLNode := compose.InvokableLambda(func(ctx context.Context, state *WorkflowState) (*WorkflowState, error) {
		sql, err := executor.GenerateSQL(ctx, state.Question, state.PreviousError)
		if err != nil {
			return nil, fmt.Errorf("generate SQL failed on attempt %d: %w", state.Attempt+1, err)
		}
		state.SQL = sql
		g.Log().Infof(ctx, "NL2SQL attempt %d/%d, SQL: %s", state.Attempt+1, maxRetries+1, sql)
		return state, nil
	})

	// Node 2: 执行 SQL（启用 callback 以支持 SQL 安全校验）
	executeSQLNode := compose.InvokableLambda(func(ctx context.Context, state *WorkflowState) (*WorkflowState, error) {
		// 检查 callback 中的 SQL 安全校验结果
		if err := checkSQLValidation(ctx); err != nil {
			errorMsg := fmt.Sprintf("Attempt %d: SQL validation rejected [%s]: %s", state.Attempt+1, state.SQL, err.Error())
			state.AllErrors = append(state.AllErrors, errorMsg)
			state.PreviousError = fmt.Sprintf("SQL: %s\nValidation Error: %s", state.SQL, err.Error())
			state.Attempt++
			g.Log().Warningf(ctx, "NL2SQL SQL validation failed on attempt %d/%d: %v", state.Attempt, maxRetries+1, err)
			state.Results = nil
			return state, nil
		}

		results, err := executor.ExecuteSQL(ctx, state.SQL)
		if err != nil {
			// 执行失败，记录错误但不返回 error（让 branch 路由决定下一步）
			errorMsg := fmt.Sprintf("Attempt %d: SQL [%s] failed with error: %s", state.Attempt+1, state.SQL, err.Error())
			state.AllErrors = append(state.AllErrors, errorMsg)
			state.PreviousError = fmt.Sprintf("SQL: %s\nError: %s", state.SQL, err.Error())
			state.Attempt++
			g.Log().Warningf(ctx, "NL2SQL execute failed on attempt %d/%d: %v", state.Attempt, maxRetries+1, err)
			// Results 置为 nil 表示失败
			state.Results = nil
			return state, nil
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

	// Node 4: 错误处理（达到最大重试次数后返回错误）
	handleErrorNode := compose.InvokableLambda(func(ctx context.Context, state *WorkflowState) (*WorkflowState, error) {
		return nil, fmt.Errorf("SQL execution failed after %d attempts. Errors:\n%s",
			maxRetries+1, strings.Join(state.AllErrors, "\n"))
	})

	// 添加节点到 Graph
	if err := graph.AddLambdaNode(nodeKeyGenerateSQL, generateSQLNode); err != nil {
		return nil, fmt.Errorf("add generate_sql node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeKeyExecuteSQL, executeSQLNode); err != nil {
		return nil, fmt.Errorf("add execute_sql node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeKeyBuildAnswer, buildAnswerNode); err != nil {
		return nil, fmt.Errorf("add build_answer node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeKeyHandleError, handleErrorNode); err != nil {
		return nil, fmt.Errorf("add handle_error node: %w", err)
	}

	// 边: START → generate_sql → execute_sql
	if err := graph.AddEdge(compose.START, nodeKeyGenerateSQL); err != nil {
		return nil, fmt.Errorf("add edge START->generate_sql: %w", err)
	}
	if err := graph.AddEdge(nodeKeyGenerateSQL, nodeKeyExecuteSQL); err != nil {
		return nil, fmt.Errorf("add edge generate_sql->execute_sql: %w", err)
	}

	// 条件分支: execute_sql 之后根据执行结果路由
	// - 成功 → build_answer
	// - 失败但未超限 → generate_sql（回环重试）
	// - 失败且已超限 → handle_error
	branch := compose.NewGraphBranch[*WorkflowState](
		func(ctx context.Context, state *WorkflowState) (string, error) {
			if state.Results != nil {
				// SQL 执行成功
				return nodeKeyBuildAnswer, nil
			}
			if state.Attempt > maxRetries {
				// 超过最大重试次数
				return nodeKeyHandleError, nil
			}
			// 重试
			return nodeKeyGenerateSQL, nil
		},
		map[string]bool{
			nodeKeyBuildAnswer: true,
			nodeKeyGenerateSQL: true,
			nodeKeyHandleError: true,
		},
	)
	if err := graph.AddBranch(nodeKeyExecuteSQL, branch); err != nil {
		return nil, fmt.Errorf("add branch after execute_sql: %w", err)
	}

	// 边: build_answer → END, handle_error → END
	if err := graph.AddEdge(nodeKeyBuildAnswer, compose.END); err != nil {
		return nil, fmt.Errorf("add edge build_answer->END: %w", err)
	}
	if err := graph.AddEdge(nodeKeyHandleError, compose.END); err != nil {
		return nil, fmt.Errorf("add handle_error->END: %w", err)
	}

	// 编译
	return graph.Compile(ctx)
}

// RunRetry 使用编译后的 Graph 执行重试工作流
func RunRetry(ctx context.Context, runnable compose.Runnable[*WorkflowState, *WorkflowState], question string) (*internalModel.AskOutput, error) {
	state := &WorkflowState{
		Question: question,
		Attempt:  0,
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
