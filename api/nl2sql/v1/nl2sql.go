package v1

import "github.com/gogf/gf/v2/frame/g"

// --- 训练相关 API ---

// TrainDDLReq 训练DDL请求
type TrainDDLReq struct {
	g.Meta `path:"/nl2sql/train/ddl" method:"post" tags:"NL2SQL" summary:"训练DDL schema"`
	DDL    string `json:"ddl" v:"required#DDL内容不能为空"`
}

// TrainDDLRes 训练DDL响应
type TrainDDLRes struct {
	ID string `json:"id"`
}

// TrainDocReq 训练文档请求
type TrainDocReq struct {
	g.Meta        `path:"/nl2sql/train/doc" method:"post" tags:"NL2SQL" summary:"训练文档"`
	Documentation string `json:"documentation" v:"required#文档内容不能为空"`
}

// TrainDocRes 训练文档响应
type TrainDocRes struct {
	ID string `json:"id"`
}

// TrainSQLReq 训练SQL示例请求
type TrainSQLReq struct {
	g.Meta   `path:"/nl2sql/train/sql" method:"post" tags:"NL2SQL" summary:"训练SQL示例"`
	Question string `json:"question" v:"required#问题不能为空"`
	SQL      string `json:"sql" v:"required#SQL语句不能为空"`
}

// TrainSQLRes 训练SQL示例响应
type TrainSQLRes struct {
	ID string `json:"id"`
}

// --- 查询相关 API ---

// AskReq 自然语言提问请求
type AskReq struct {
	g.Meta       `path:"/nl2sql/ask" method:"post" tags:"NL2SQL" summary:"自然语言提问"`
	Question     string `json:"question" v:"required#问题不能为空"`
	WorkflowType string `json:"workflowType" d:"simple"` // simple, retry 或 agent
}

// AskRes 自然语言提问响应
type AskRes struct {
	Question string      `json:"question"`
	SQL      string      `json:"sql"`
	Results  interface{} `json:"results"`
	Answer   string      `json:"answer"`
}

// --- 训练数据管理 API ---

// RemoveTrainingDataReq 删除训练数据请求
type RemoveTrainingDataReq struct {
	g.Meta `path:"/nl2sql/training-data/{id}" method:"delete" tags:"NL2SQL" summary:"删除训练数据"`
	Id     string `json:"id" in:"path" v:"required#ID不能为空"`
}

// RemoveTrainingDataRes 删除训练数据响应
type RemoveTrainingDataRes struct{}

// ListTrainingDataReq 列出训练数据请求
type ListTrainingDataReq struct {
	g.Meta `path:"/nl2sql/training-data" method:"get" tags:"NL2SQL" summary:"列出训练数据"`
	Type   string `json:"type" in:"query"` // ddl/doc/sql，留空返回全部
}

// ListTrainingDataRes 列出训练数据响应
type ListTrainingDataRes struct {
	List []TrainingDataItem `json:"list"`
}

// TrainingDataItem 训练数据条目
type TrainingDataItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Question string `json:"question,omitempty"`
}
