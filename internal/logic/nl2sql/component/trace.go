package component

import (
	"context"
	"fmt"

	cozeloopCb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	langfuseCb "github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
	"github.com/gogf/gf/v2/frame/g"
)

// InitTrace 根据配置初始化 Trace 回调
// 支持 langfuse 和 cozeloop 两种 trace 后端
// 返回一个 cleanup 函数，用于在服务关闭时刷新并清理 trace 数据
func InitTrace(ctx context.Context) (cleanup func(), err error) {
	cfg := g.Cfg()
	provider := cfg.MustGet(ctx, "nl2sql.trace.provider").String()
	if provider == "" {
		g.Log().Info(ctx, "No trace provider configured, tracing is disabled")
		return func() {}, nil
	}

	switch provider {
	case "langfuse":
		return initLangfuse(ctx)
	case "cozeloop":
		return initCozeloop(ctx)
	default:
		return nil, fmt.Errorf("unsupported trace provider: %s, supported: langfuse/cozeloop", provider)
	}
}

// initLangfuse 初始化 Langfuse trace
func initLangfuse(ctx context.Context) (func(), error) {
	cfg := g.Cfg()
	host := cfg.MustGet(ctx, "nl2sql.trace.langfuse.host").String()
	publicKey := cfg.MustGet(ctx, "nl2sql.trace.langfuse.publicKey").String()
	secretKey := cfg.MustGet(ctx, "nl2sql.trace.langfuse.secretKey").String()

	if host == "" || publicKey == "" || secretKey == "" {
		return nil, fmt.Errorf("langfuse trace requires host, publicKey, and secretKey in config")
	}

	handler, flusher := langfuseCb.NewLangfuseHandler(&langfuseCb.Config{
		Host:      host,
		PublicKey: publicKey,
		SecretKey: secretKey,
		Name:      "nl2sql",
	})

	callbacks.AppendGlobalHandlers(handler)
	g.Log().Info(ctx, "Langfuse trace initialized successfully")

	return flusher, nil
}

// initCozeloop 初始化 CozeLoop trace
func initCozeloop(ctx context.Context) (func(), error) {
	// CozeLoop 通过环境变量配置:
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token
	client, err := cozeloop.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create cozeloop client: %w", err)
	}

	handler := cozeloopCb.NewLoopHandler(client)
	callbacks.AppendGlobalHandlers(handler)
	g.Log().Info(ctx, "CozeLoop trace initialized successfully")

	cleanup := func() {
		client.Close(context.Background())
	}
	return cleanup, nil
}
