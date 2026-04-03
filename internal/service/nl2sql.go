// ==========================================================================
// Code generated and target to be maintained by GoFrame CLI tool.
// You can modify this file to define the service interface for nl2sql.
// ==========================================================================

package service

import (
	"context"

	"nl2sql/internal/model"
)

// INl2sql NL2SQL 服务接口
type INl2sql interface {
	// Init 初始化服务（Trace + 组件 + 工作流），返回 cleanup 函数
	Init(ctx context.Context) (func(), error)
	// Ask 自然语言提问
	Ask(ctx context.Context, input *model.AskInput) (*model.AskOutput, error)
	// TrainDDL 训练 DDL schema
	TrainDDL(ctx context.Context, ddl string) (*model.TrainOutput, error)
	// TrainDoc 训练文档
	TrainDoc(ctx context.Context, documentation string) (*model.TrainOutput, error)
	// TrainSQL 训练 SQL 示例（问题-SQL对）
	TrainSQL(ctx context.Context, question, sql string) (*model.TrainOutput, error)
	// RemoveTrainingData 删除训练数据
	RemoveTrainingData(ctx context.Context, id string) error
	// ListTrainingData 列出训练数据
	ListTrainingData(ctx context.Context, dataType string) ([]*model.TrainingDataItem, error)
}

var localNl2sql INl2sql

// Nl2sql 获取 NL2SQL 服务实例
func Nl2sql() INl2sql {
	if localNl2sql == nil {
		panic("service INl2sql is not registered. Please make sure to import the logic package: import _ \"nl2sql/internal/logic/nl2sql\"")
	}
	return localNl2sql
}

// RegisterNl2sql 注册 NL2SQL 服务实现
func RegisterNl2sql(s INl2sql) {
	localNl2sql = s
}
