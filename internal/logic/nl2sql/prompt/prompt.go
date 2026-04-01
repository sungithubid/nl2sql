package prompt

import (
	"context"
	"fmt"
	"sync"

	einoPrompt "github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// Registry 提示词模板注册表
// 使用 eino ChatTemplate 作为底层引擎
type Registry struct {
	mu        sync.RWMutex
	templates map[string]*einoPrompt.DefaultChatTemplate
}

// NewRegistry 创建一个新的模板注册表
func NewRegistry() *Registry {
	return &Registry{
		templates: make(map[string]*einoPrompt.DefaultChatTemplate),
	}
}

// Register 注册一个 ChatTemplate（如已存在则覆盖）
func (r *Registry) Register(key string, tmpl *einoPrompt.DefaultChatTemplate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[key] = tmpl
}

// Format 获取指定 key 的 ChatTemplate 并用变量渲染，返回 []*schema.Message
func (r *Registry) Format(ctx context.Context, key string, vars map[string]any) ([]*schema.Message, error) {
	r.mu.RLock()
	tmpl, ok := r.templates[key]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("prompt template %q not found", key)
	}
	return tmpl.Format(ctx, vars)
}

// --- 包级默认实例 ---

var defaultRegistry = NewRegistry()

// Register 注册 ChatTemplate 到默认注册表
func Register(key string, tmpl *einoPrompt.DefaultChatTemplate) {
	defaultRegistry.Register(key, tmpl)
}

// Format 使用默认注册表渲染模板，返回 []*schema.Message
func Format(ctx context.Context, key string, vars map[string]any) ([]*schema.Message, error) {
	return defaultRegistry.Format(ctx, key, vars)
}

// TODO: LoadFromDB 从数据库加载模板，覆盖默认硬编码模板
// func LoadFromDB(ctx context.Context, db gdb.DB) error {
//     // 示例：从 prompt_templates 表加载
//     // rows, err := db.Model("prompt_templates").All()
//     // for _, row := range rows {
//     //     tmpl := einoPrompt.FromMessages(schema.FString,
//     //         schema.SystemMessage(row["system_content"].String()),
//     //         schema.UserMessage(row["user_content"].String()),
//     //     )
//     //     Register(row["key"].String(), tmpl)
//     // }
//     return nil
// }

// TODO: LoadFromCozeLoop 从 CozeLoop 工程化平台加载模板
// func LoadFromCozeLoop(ctx context.Context, workspaceID string) error {
//     // 示例：通过 CozeLoop API 拉取 prompt 模板
//     // prompts, err := cozeloop.ListPrompts(ctx, workspaceID)
//     // for _, p := range prompts {
//     //     tmpl := einoPrompt.FromMessages(schema.FString,
//     //         schema.SystemMessage(p.SystemContent),
//     //         schema.UserMessage(p.UserContent),
//     //     )
//     //     Register(p.Key, tmpl)
//     // }
//     return nil
// }
