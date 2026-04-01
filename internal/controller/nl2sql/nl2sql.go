package nl2sql

import (
	"context"

	v1 "nl2sql/api/nl2sql/v1"
	"nl2sql/internal/model"
	"nl2sql/internal/service"
)

// ControllerV1 NL2SQL 控制器
type ControllerV1 struct{}

// NewV1 创建 NL2SQL 控制器实例
func NewV1() *ControllerV1 {
	return &ControllerV1{}
}

// TrainDDL 训练 DDL schema
func (c *ControllerV1) TrainDDL(ctx context.Context, req *v1.TrainDDLReq) (res *v1.TrainDDLRes, err error) {
	output, err := service.Nl2sql().TrainDDL(ctx, req.DDL)
	if err != nil {
		return nil, err
	}
	return &v1.TrainDDLRes{ID: output.ID}, nil
}

// TrainDoc 训练文档
func (c *ControllerV1) TrainDoc(ctx context.Context, req *v1.TrainDocReq) (res *v1.TrainDocRes, err error) {
	output, err := service.Nl2sql().TrainDoc(ctx, req.Documentation)
	if err != nil {
		return nil, err
	}
	return &v1.TrainDocRes{ID: output.ID}, nil
}

// TrainSQL 训练 SQL 示例
func (c *ControllerV1) TrainSQL(ctx context.Context, req *v1.TrainSQLReq) (res *v1.TrainSQLRes, err error) {
	output, err := service.Nl2sql().TrainSQL(ctx, req.Question, req.SQL)
	if err != nil {
		return nil, err
	}
	return &v1.TrainSQLRes{ID: output.ID}, nil
}

// Ask 自然语言提问
func (c *ControllerV1) Ask(ctx context.Context, req *v1.AskReq) (res *v1.AskRes, err error) {
	output, err := service.Nl2sql().Ask(ctx, &model.AskInput{
		Question:     req.Question,
		WorkflowType: req.WorkflowType,
	})
	if err != nil {
		return nil, err
	}
	return &v1.AskRes{
		Question: output.Question,
		SQL:      output.SQL,
		Results:  output.Results,
		Answer:   output.Answer,
	}, nil
}

// RemoveTrainingData 删除训练数据
func (c *ControllerV1) RemoveTrainingData(ctx context.Context, req *v1.RemoveTrainingDataReq) (res *v1.RemoveTrainingDataRes, err error) {
	if err = service.Nl2sql().RemoveTrainingData(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.RemoveTrainingDataRes{}, nil
}

// ListTrainingData 列出训练数据
func (c *ControllerV1) ListTrainingData(ctx context.Context, req *v1.ListTrainingDataReq) (res *v1.ListTrainingDataRes, err error) {
	items, err := service.Nl2sql().ListTrainingData(ctx, req.Type)
	if err != nil {
		return nil, err
	}

	list := make([]v1.TrainingDataItem, 0, len(items))
	for _, item := range items {
		list = append(list, v1.TrainingDataItem{
			ID:       item.ID,
			Type:     item.Type,
			Content:  item.Content,
			Question: item.Question,
		})
	}

	return &v1.ListTrainingDataRes{List: list}, nil
}
