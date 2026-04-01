package component

import (
	"context"
	"fmt"

	embOpenAI "github.com/cloudwego/eino-ext/components/embedding/openai"
	einoOpenAI "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	qdrant "github.com/qdrant/go-client/qdrant"
)

// Components 包含所有 eino 组件的集合
type Components struct {
	ChatModel   model.ChatModel
	Embedder    embedding.Embedder
	Client      *qdrant.Client // Qdrant gRPC 客户端
	VectorStore *VectorStore   // Qdrant 向量存储封装
	TargetDB    gdb.DB         // 目标业务数据库（连接池单例）
}

// NewComponents 根据配置创建所有 eino 组件
func NewComponents(ctx context.Context) (*Components, error) {
	chatModel, err := NewChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	embedder, err := NewEmbedding(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	client, err := NewQdrantClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	cfg := g.Cfg()
	dimension := cfg.MustGet(ctx, "nl2sql.embedding.dimensions", 1536).Int()
	vectorStore := NewVectorStore(client, embedder, dimension)

	targetDB, err := NewTargetDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create target database: %w", err)
	}

	return &Components{
		ChatModel:   chatModel,
		Embedder:    embedder,
		Client:      client,
		VectorStore: vectorStore,
		TargetDB:    targetDB,
	}, nil
}

// NewChatModel 根据配置创建 ChatModel
func NewChatModel(ctx context.Context) (model.ChatModel, error) {
	cfg := g.Cfg()
	provider := cfg.MustGet(ctx, "nl2sql.llm.provider", "openai").String()
	apiKey := cfg.MustGet(ctx, "nl2sql.llm.apiKey").String()
	modelName := cfg.MustGet(ctx, "nl2sql.llm.model", "gpt-4o").String()
	baseURL := cfg.MustGet(ctx, "nl2sql.llm.baseUrl").String()
	temperature := cfg.MustGet(ctx, "nl2sql.llm.temperature", 0.0).Float32()

	switch provider {
	case "openai", "deepseek", "qwen", "openrouter":
		// 这些厂商都兼容 OpenAI 接口，使用 openai ChatModel + 自定义 baseUrl
		config := &einoOpenAI.ChatModelConfig{
			APIKey:      apiKey,
			Model:       modelName,
			Temperature: &temperature,
		}
		if baseURL != "" {
			config.BaseURL = baseURL
		}
		// 为常用厂商设置默认 baseUrl
		if baseURL == "" {
			switch provider {
			case "deepseek":
				config.BaseURL = "https://api.deepseek.com/v1"
			case "qwen":
				config.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
			case "openrouter":
				config.BaseURL = "https://openrouter.ai/api/v1"
			}
		}
		return einoOpenAI.NewChatModel(ctx, config)
	case "ollama":
		// Ollama 也兼容 OpenAI 接口
		ollamaBaseURL := baseURL
		if ollamaBaseURL == "" {
			ollamaBaseURL = "http://localhost:11434/v1"
		}
		config := &einoOpenAI.ChatModelConfig{
			Model:       modelName,
			BaseURL:     ollamaBaseURL,
			Temperature: &temperature,
		}
		if apiKey != "" {
			config.APIKey = apiKey
		} else {
			config.APIKey = "ollama" // ollama 不需要 apiKey，但字段必填
		}
		return einoOpenAI.NewChatModel(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s, supported: openai/deepseek/qwen/ollama/openrouter", provider)
	}
}

// NewEmbedding 根据配置创建 Embedding
func NewEmbedding(ctx context.Context) (embedding.Embedder, error) {
	cfg := g.Cfg()
	provider := cfg.MustGet(ctx, "nl2sql.embedding.provider", "openai").String()
	apiKey := cfg.MustGet(ctx, "nl2sql.embedding.apiKey").String()
	modelName := cfg.MustGet(ctx, "nl2sql.embedding.model", "text-embedding-3-small").String()
	baseURL := cfg.MustGet(ctx, "nl2sql.embedding.baseUrl").String()

	switch provider {
	case "openai":
		config := &embOpenAI.EmbeddingConfig{
			APIKey: apiKey,
			Model:  modelName,
		}
		if baseURL != "" {
			config.BaseURL = baseURL
		}
		return embOpenAI.NewEmbedder(ctx, config)
	case "ollama":
		ollamaBaseURL := baseURL
		if ollamaBaseURL == "" {
			ollamaBaseURL = "http://localhost:11434/v1"
		}
		config := &embOpenAI.EmbeddingConfig{
			Model:   modelName,
			BaseURL: ollamaBaseURL,
		}
		if apiKey != "" {
			config.APIKey = apiKey
		} else {
			config.APIKey = "ollama"
		}
		return embOpenAI.NewEmbedder(ctx, config)
	default:
		// 其他厂商也可通过 OpenAI 兼容接口使用
		config := &embOpenAI.EmbeddingConfig{
			APIKey: apiKey,
			Model:  modelName,
		}
		if baseURL != "" {
			config.BaseURL = baseURL
		}
		return embOpenAI.NewEmbedder(ctx, config)
	}
}

// NewQdrantClient 创建 Qdrant gRPC 客户端
func NewQdrantClient(ctx context.Context) (*qdrant.Client, error) {
	cfg := g.Cfg()
	host := cfg.MustGet(ctx, "nl2sql.qdrant.host", "localhost").String()
	port := cfg.MustGet(ctx, "nl2sql.qdrant.port", 6334).Int()

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect qdrant at %s:%d: %w", host, port, err)
	}
	return client, nil
}

// CollectionName 获取带前缀的 collection 名称
func CollectionName(ctx context.Context, suffix string) string {
	cfg := g.Cfg()
	prefix := cfg.MustGet(ctx, "nl2sql.qdrant.collectionPrefix", "nl2sql").String()
	return prefix + "_" + suffix
}

// NewTargetDB 根据配置创建目标业务数据库连接（连接池单例）
func NewTargetDB(ctx context.Context) (gdb.DB, error) {
	cfg := g.Cfg()
	link := cfg.MustGet(ctx, "nl2sql.datasource.link").String()
	if link == "" {
		return nil, fmt.Errorf("nl2sql.datasource.link is not configured")
	}

	db, err := gdb.New(gdb.ConfigNode{
		Link: link,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create target database connection: %w", err)
	}
	return db, nil
}
