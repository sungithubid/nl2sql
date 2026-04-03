package trace

import (
	"context"
	"fmt"

	cozeloopCb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	langfuseCb "github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
	"github.com/gogf/gf/v2/frame/g"
)

// Config trace 配置（调用方从各自的配置命名空间读取后传入）
type Config struct {
	Provider string // langfuse 或 cozeloop
	Langfuse LangfuseConfig
}

// LangfuseConfig Langfuse trace 配置
type LangfuseConfig struct {
	Host      string
	PublicKey string
	SecretKey string
}

// Init 根据 config 初始化 Trace 回调
// 支持 langfuse 和 cozeloop 两种 trace 后端
// 返回一个 cleanup 函数，用于在服务关闭时刷新并清理 trace 数据
func Init(ctx context.Context, cfg *Config) (func(), error) {
	if cfg == nil || cfg.Provider == "" {
		g.Log().Info(ctx, "No trace provider configured, tracing is disabled")
		return func() {}, nil
	}

	switch cfg.Provider {
	case "langfuse":
		return initLangfuse(ctx, &cfg.Langfuse)
	case "cozeloop":
		return initCozeloop(ctx)
	default:
		return nil, fmt.Errorf("unsupported trace provider: %s, supported: langfuse/cozeloop", cfg.Provider)
	}
}

// initLangfuse 初始化 Langfuse trace
func initLangfuse(ctx context.Context, cfg *LangfuseConfig) (func(), error) {
	if cfg.Host == "" || cfg.PublicKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("langfuse trace requires host, publicKey, and secretKey")
	}

	handler, flusher := langfuseCb.NewLangfuseHandler(&langfuseCb.Config{
		Host:      cfg.Host,
		PublicKey: cfg.PublicKey,
		SecretKey: cfg.SecretKey,
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
