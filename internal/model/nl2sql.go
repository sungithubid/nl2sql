package model

// TrainDDLInput 训练DDL的输入
type TrainDDLInput struct {
	DDL string // DDL 语句内容
}

// TrainDocInput 训练文档的输入
type TrainDocInput struct {
	Documentation string // 文档内容
}

// TrainSQLInput 训练SQL示例的输入
type TrainSQLInput struct {
	Question string // 自然语言问题
	SQL      string // 对应的SQL语句
}

// AskInput 自然语言提问的输入
type AskInput struct {
	Question     string // 自然语言问题
	WorkflowType string // 工作流类型: simple 或 retry
}

// AskOutput 提问的输出
type AskOutput struct {
	Question string      `json:"question"` // 原始问题
	SQL      string      `json:"sql"`      // 生成的SQL
	Results  interface{} `json:"results"`  // 查询结果
	Answer   string      `json:"answer"`   // 自然语言回答
}

// TrainOutput 训练操作的输出
type TrainOutput struct {
	ID string `json:"id"` // 训练数据ID
}

// TrainingDataItem 训练数据项
type TrainingDataItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"`     // ddl, doc, sql
	Content  string `json:"content"`  // 主要内容
	Question string `json:"question"` // 问题（仅SQL类型有）
}
