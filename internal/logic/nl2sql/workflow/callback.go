package workflow

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/gogf/gf/v2/frame/g"

	"nl2sql/internal/logic/nl2sql/security"
)

// sqlValidationKey 用于在 context 中传递 SQL 校验结果
type sqlValidationKey struct{}

// newSQLValidationHandler 创建 Eino callback handler，在 execute_sql 节点执行前校验 SQL
// 使用方式：compose.WithCallbacks(handler).DesignateNode("execute_sql")
func newSQLValidationHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			state, ok := input.(*WorkflowState)
			if !ok || state.SQL == "" {
				return ctx
			}

			result := security.ValidateSQL(state.SQL)
			if !result.Valid {
				g.Log().Warningf(ctx, "SQL validation rejected: %s, SQL: %s", result.Reason, state.SQL)
				ctx = context.WithValue(ctx, sqlValidationKey{}, &result)
			}
			return ctx
		}).
		Build()
}

// checkSQLValidation 从 context 中获取 callback 校验结果
// 返回 nil 表示校验通过，非 nil 表示 SQL 被拒绝
func checkSQLValidation(ctx context.Context) error {
	val, _ := ctx.Value(sqlValidationKey{}).(*security.SQLValidationResult)
	if val != nil && !val.Valid {
		return fmt.Errorf("SQL validation failed: %s", val.Reason)
	}
	return nil
}
