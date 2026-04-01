package workflow

// WorkflowState 工作流各节点之间的统一数据载体
// 使用统一的 State 结构使 Eino Graph/Chain 各节点具有相同的 I/O 类型签名
type WorkflowState struct {
	Question      string                   // 用户的自然语言问题
	PreviousError string                   // 上次 SQL 执行的错误信息（用于重试时传递给 LLM 修正）
	SQL           string                   // 生成的 SQL
	Results       []map[string]interface{} // SQL 执行结果
	Answer        string                   // 自然语言回答
	Attempt       int                      // 当前重试次数
	AllErrors     []string                 // 所有重试中的错误记录
}
