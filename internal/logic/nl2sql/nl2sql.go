package nl2sql

import (
	"context"
	"fmt"
	"sync"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/gogf/gf/v2/frame/g"

	"nl2sql/internal/logic/nl2sql/component"
	"nl2sql/internal/logic/nl2sql/workflow"
	"nl2sql/internal/model"
	"nl2sql/internal/service"
)

// sNl2sql NL2SQL 服务实现（GoFrame 命名约定: s + 首字母大写）
type sNl2sql struct {
	comp           *component.Components
	mu             sync.RWMutex
	simpleRunnable compose.Runnable[*workflow.WorkflowState, *workflow.WorkflowState]
	retryRunnable  compose.Runnable[*workflow.WorkflowState, *workflow.WorkflowState]
	agentInstance  *react.Agent
}

// init 注册 NL2SQL 服务实现到 service 层
func init() {
	service.RegisterNl2sql(&sNl2sql{})
}

// Init 初始化 NL2SQL 服务（创建所有 eino 组件）
// 在 cmd 启动时调用，用于建立与外部服务的连接
func Init(ctx context.Context) error {
	s := service.Nl2sql().(*sNl2sql)
	comp, err := component.NewComponents(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize NL2SQL components: %w", err)
	}
	s.comp = comp

	// 预编译 Simple 工作流（Chain）
	simpleRunnable, err := workflow.CompileSimpleChain(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to compile simple workflow: %w", err)
	}
	s.simpleRunnable = simpleRunnable

	// 预编译 Retry 工作流（Graph）
	retryRunnable, err := workflow.CompileRetryGraph(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to compile retry workflow: %w", err)
	}
	s.retryRunnable = retryRunnable

	// 创建 React Agent（使用独立的 ChatModel，避免 BindTools 污染共享实例）
	agentChatModel, err := component.NewChatModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to create agent chat model: %w", err)
	}
	agentInst, err := workflow.CompileAgent(ctx, s, agentChatModel)
	if err != nil {
		return fmt.Errorf("failed to compile react agent: %w", err)
	}
	s.agentInstance = agentInst

	g.Log().Info(ctx, "NL2SQL service components and workflows initialized successfully")
	return nil
}

// Ask 自然语言提问入口
func (s *sNl2sql) Ask(ctx context.Context, input *model.AskInput) (*model.AskOutput, error) {
	if s.comp == nil {
		return nil, fmt.Errorf("NL2SQL service not initialized, please check configuration")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	switch input.WorkflowType {
	case "agent":
		return workflow.RunAgent(ctx, s.agentInstance, input.Question)
	case "retry":
		return workflow.RunRetry(ctx, s.retryRunnable, input.Question)
	case "simple", "":
		return workflow.RunSimple(ctx, s.simpleRunnable, input.Question)
	default:
		return nil, fmt.Errorf("unsupported workflow type: %s, supported: simple/retry/agent", input.WorkflowType)
	}
}

// GetChatModel 返回 ChatModel 组件（实现 workflow.Executor 接口）
func (s *sNl2sql) GetChatModel() einoModel.ChatModel {
	return s.comp.ChatModel
}

// TrainDDL 训练 DDL schema
func (s *sNl2sql) TrainDDL(ctx context.Context, ddl string) (*model.TrainOutput, error) {
	if s.comp == nil {
		return nil, fmt.Errorf("NL2SQL service not initialized")
	}
	id, err := s.trainDDL(ctx, ddl)
	if err != nil {
		return nil, err
	}
	return &model.TrainOutput{ID: id}, nil
}

// TrainDoc 训练文档
func (s *sNl2sql) TrainDoc(ctx context.Context, documentation string) (*model.TrainOutput, error) {
	if s.comp == nil {
		return nil, fmt.Errorf("NL2SQL service not initialized")
	}
	id, err := s.trainDoc(ctx, documentation)
	if err != nil {
		return nil, err
	}
	return &model.TrainOutput{ID: id}, nil
}

// TrainSQL 训练 SQL 示例
func (s *sNl2sql) TrainSQL(ctx context.Context, question, sql string) (*model.TrainOutput, error) {
	if s.comp == nil {
		return nil, fmt.Errorf("NL2SQL service not initialized")
	}
	id, err := s.trainSQL(ctx, question, sql)
	if err != nil {
		return nil, err
	}
	return &model.TrainOutput{ID: id}, nil
}

// RemoveTrainingData 删除训练数据
func (s *sNl2sql) RemoveTrainingData(ctx context.Context, id string) error {
	if s.comp == nil {
		return fmt.Errorf("NL2SQL service not initialized")
	}

	collections := []string{
		component.CollectionName(ctx, "ddl"),
		component.CollectionName(ctx, "doc"),
		component.CollectionName(ctx, "sql"),
	}

	for _, collection := range collections {
		_ = s.comp.VectorStore.Delete(ctx, collection, id)
	}
	return nil
}

// ListTrainingData 列出训练数据
func (s *sNl2sql) ListTrainingData(ctx context.Context, dataType string) ([]*model.TrainingDataItem, error) {
	if s.comp == nil {
		return nil, fmt.Errorf("NL2SQL service not initialized")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []*model.TrainingDataItem
	types := []string{"ddl", "doc", "sql"}

	if dataType != "" {
		types = []string{dataType}
	}

	for _, t := range types {
		collection := component.CollectionName(ctx, t)
		collectionItems, err := s.listFromCollection(ctx, collection, t)
		if err != nil {
			continue
		}
		items = append(items, collectionItems...)
	}

	return items, nil
}

// listFromCollection 从指定 collection 列出数据
func (s *sNl2sql) listFromCollection(ctx context.Context, collection string, dataType string) ([]*model.TrainingDataItem, error) {
	docs, err := s.comp.VectorStore.List(ctx, collection, 100)
	if err != nil {
		return nil, err
	}

	items := make([]*model.TrainingDataItem, 0, len(docs))
	for _, doc := range docs {
		item := &model.TrainingDataItem{
			ID:      doc.ID,
			Type:    dataType,
			Content: doc.Content,
		}
		if dataType == "sql" && doc.MetaData != nil {
			if q, ok := doc.MetaData["question"]; ok {
				if qs, ok := q.(string); ok {
					item.Question = qs
				}
			}
		}
		items = append(items, item)
	}

	return items, nil
}
