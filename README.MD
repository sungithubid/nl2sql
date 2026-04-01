# NL2SQL Backend Service

基于 [GoFrame](https://goframe.org/) + [cloudwego/eino](https://github.com/cloudwego/eino) 构建的 NL2SQL（自然语言转SQL）后端服务示例项目，灵感来源于 Python [Vanna](https://github.com/vanna-ai/vanna) 框架。

## 特性

- **多种 LLM 支持** — OpenAI / DeepSeek / Qwen / Ollama / OpenRouter，通过配置切换
- **Qdrant 向量数据库** — 自封装 VectorStore 层（基于 qdrant/go-client），支持预计算向量并发检索
- **RAG 架构** — 训练 DDL Schema、文档和 SQL 示例，提升 SQL 生成准确率
- **三种工作流模式**
  - `simple`：generate → execute → answer
  - `retry`：generate → execute（失败自动重试最多 3 次）→ answer
  - `agent`：**React Agent 模式** — AI 自主决策，通过工具调用完成检索、生成、执行全流程
- **Agent Trace** — 支持 [Langfuse](https://langfuse.com/) 和 [CozeLoop](https://github.com/coze-dev/cozeloop-go) 可观测性
- **严格的 SQL 安全校验** — 基于 AST (sqlparser) 的多层安全防御，结合 Eino Callback 切面拦截多语句注入与高危操作，确保物理库仅执行安全 SELECT 查询
- **GoFrame 标准架构** — 遵循 GoFrame 分层架构，自带 Swagger API 文档

## 架构

```
┌──────────────────────────────────────────────────────────┐
│                    HTTP API (GoFrame)                     │
│  POST /api/v1/nl2sql/train/ddl                           │
│  POST /api/v1/nl2sql/train/doc                           │
│  POST /api/v1/nl2sql/train/sql                           │
│  POST /api/v1/nl2sql/ask                                 │
│  GET  /api/v1/nl2sql/training-data                       │
│  DELETE /api/v1/nl2sql/training-data/{id}                │
├──────────────────────────────────────────────────────────┤
│                   Controller Layer                        │
├──────────────────────────────────────────────────────────┤
│                    NL2SQL Service                         │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌─────────────┐ │
│  │ Training │ │ Generate │ │ Execute  │ │ BuildAnswer │ │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬──────┘ │
│       │            │            │               │        │
│  ┌────▼────────────▼────────────▼───────────────▼──────┐ │
│  │              Eino Components + VectorStore           │ │
│  │  ChatModel │ Embedding │ VectorStore                 │ │
│  └──────┬──────────┬──────────┬────────────────────────┘ │
│         │          │          │                          │
│   ┌─────▼──────────▼──────────▼────────────────────────┐ │
│   │           Trace (Langfuse / CozeLoop)               │ │
│   └────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────┤
│     LLM API      Embedding    Qdrant     Target DB       │
│  (Multi-Provider)   API     (Vector DB)  (Business)      │
└──────────────────────────────────────────────────────────┘
```

### NL2SQL Workflows

**Workflow 1 — Simple**
```
start → generate_sql → execute_sql → build_answer → end
```

**Workflow 2 — Retry**
```
start → generate_sql → execute_sql ─── success ──→ build_answer → end
                 ▲           │
                 │         error
                 │           │
                 └───── retry (max 3) ──→ return error
```

**Workflow 3 — Agent (React Agent)**
```
User Question
     ↓
┌─────────────────────────────────────┐
│          React Agent (LLM)          │
│  自主决策调用哪些工具、调用顺序       │
│                                     │
│  可用工具:                           │
│  ├─ retrieve_ddl     检索DDL Schema  │
│  ├─ retrieve_docs    检索业务文档    │
│  ├─ retrieve_sql_examples 检索SQL示例│
│  ├─ generate_sql     生成SQL         │
│  └─ execute_sql      执行SQL         │
│                                     │
│  循环: Reason → Act → Observe       │
│  直到获得最终答案                    │
└─────────────────────────────────────┘
     ↓
Natural Language Answer
```

## 项目结构

```
nl2sql/
├── api/nl2sql/v1/
│   └── nl2sql.go                          # API 定义（6个端点）
├── internal/
│   ├── cmd/cmd.go                          # 路由注册 + 服务初始化
│   ├── controller/nl2sql/
│   │   ├── nl2sql.go                       # HTTP 控制器
│   │   └── nl2sql_err.go                   # 错误定义
│   ├── logic/nl2sql/
│   │   ├── nl2sql.go                       # 主服务入口
│   │   ├── training.go                     # 训练逻辑（DDL/Doc/SQL）
│   │   ├── generate.go                     # RAG SQL 生成
│   │   ├── execute.go                      # SQL 执行
│   │   ├── answer.go                       # 自然语言回答构建
│   │   ├── component/
│   │   │   ├── factory.go                  # Eino 组件工厂
│   │   │   ├── vectorstore.go              # Qdrant 向量存储封装
│   │   │   └── trace.go                    # Trace 初始化（Langfuse/CozeLoop）
│   │   ├── prompt/
│   │   │   ├── prompt.go                   # Eino Prompt 管理组件
│   │   │   └── templates.go                # Prompt 模板定义
│   │   ├── security/
│   │   │   └── validator.go                # SQL 安全校验核心逻辑
│   │   └── workflow/
│   │       ├── simple.go                   # 简单工作流
│   │       ├── retry.go                    # 重试工作流
│   │       ├── agent.go                    # React Agent 工作流
│   │       ├── tools.go                    # Agent 工具定义
│   │       ├── state.go                    # 工作流状态
│   │       └── callback.go                 # Eino SQL安全拦截回调
│   └── model/
│       └── nl2sql.go                       # 数据模型
├── manifest/config/
│   └── config.yaml                         # 应用配置
└── go.mod
```

## 快速开始

### 1. 前置依赖

- Go 1.24+
- [Qdrant](https://qdrant.tech/) 向量数据库
- LLM API Key（OpenAI/DeepSeek/Qwen等任一）

### 2. 启动 Qdrant

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### 3. 配置

编辑 `manifest/config/config.yaml`，填写以下关键配置：

```yaml
nl2sql:
  llm:
    provider: "openai"        # 或 deepseek/qwen/ollama/openrouter
    model: "gpt-4o"
    apiKey: "sk-xxx"
  embedding:
    provider: "openai"
    model: "text-embedding-3-small"
    apiKey: "sk-xxx"
  qdrant:
    host: "localhost"
    port: 6334
  datasource:
    link: "mysql:root:password@tcp(127.0.0.1:3306)/your_database"
```

### 4. 启动服务

```bash
go run main.go
```

访问 Swagger 文档: http://localhost:8000/swagger

### 5. 训练

```bash
# 训练 DDL（数据库表结构）
curl -X POST http://localhost:8000/api/v1/nl2sql/train/ddl \
  -H 'Content-Type: application/json' \
  -d '{"ddl": "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), email VARCHAR(100), created_at DATETIME)"}'

# 训练文档（业务说明）
curl -X POST http://localhost:8000/api/v1/nl2sql/train/doc \
  -H 'Content-Type: application/json' \
  -d '{"documentation": "users表存储所有注册用户信息，created_at为注册时间"}'

# 训练 SQL 示例（问题-SQL对）
curl -X POST http://localhost:8000/api/v1/nl2sql/train/sql \
  -H 'Content-Type: application/json' \
  -d '{"question": "查询最近7天注册的用户", "sql": "SELECT * FROM users WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY) LIMIT 100"}'
```

### 6. 提问

```bash
# Simple 模式
curl -X POST http://localhost:8000/api/v1/nl2sql/ask \
  -H 'Content-Type: application/json' \
  -d '{"question": "有多少个用户？", "workflowType": "simple"}'

# Retry 模式（SQL执行出错时自动重试）
curl -X POST http://localhost:8000/api/v1/nl2sql/ask \
  -H 'Content-Type: application/json' \
  -d '{"question": "查询每个月的新注册用户数", "workflowType": "retry"}'

# Agent 模式（AI自主决策工具调用流程）
curl -X POST http://localhost:8000/api/v1/nl2sql/ask \
  -H 'Content-Type: application/json' \
  -d '{"question": "查询最近7天每天的注册用户数，按日期排序", "workflowType": "agent"}'
```

**响应示例：**

```json
{
  "code": 0,
  "message": "",
  "data": {
    "question": "有多少个用户？",
    "sql": "SELECT COUNT(*) as total_users FROM users",
    "results": [{"total_users": 1234}],
    "answer": "根据查询结果，系统中目前共有 1,234 个注册用户。"
  }
}
```

## Trace 可观测性

支持 **Langfuse** 和 **CozeLoop** 两种 trace 后端，通过 eino 的 `callbacks.AppendGlobalHandlers` 实现，所有 eino 组件调用（ChatModel、Embedding）自动上报 trace 数据。

### Langfuse

```yaml
nl2sql:
  trace:
    provider: "langfuse"
    langfuse:
      host: "https://cloud.langfuse.com"
      publicKey: "pk-lf-xxx"
      secretKey: "sk-lf-xxx"
```

### CozeLoop

CozeLoop 通过环境变量配置：

```bash
export COZELOOP_WORKSPACE_ID=your_workspace_id
export COZELOOP_API_TOKEN=your_token
```

```yaml
nl2sql:
  trace:
    provider: "cozeloop"
```

## API 参考

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/nl2sql/train/ddl` | 训练 DDL Schema |
| `POST` | `/api/v1/nl2sql/train/doc` | 训练业务文档 |
| `POST` | `/api/v1/nl2sql/train/sql` | 训练 SQL 示例（问题-SQL对） |
| `POST` | `/api/v1/nl2sql/ask` | 自然语言提问（支持 `simple`/`retry`/`agent` 工作流） |
| `GET` | `/api/v1/nl2sql/training-data` | 列出训练数据（可按 `type` 过滤） |
| `DELETE` | `/api/v1/nl2sql/training-data/{id}` | 删除训练数据 |

## 组件依赖

### Eino 组件

使用 [cloudwego/eino-ext](https://github.com/cloudwego/eino-ext) 的官方组件：

| Component | Package | Description |
|-----------|---------|-------------|
| ChatModel | `eino-ext/components/model/openai` | 多厂商 LLM（OpenAI 兼容接口） |
| Embedding | `eino-ext/components/embedding/openai` | 文本向量化 |
| Trace | `eino-ext/callbacks/langfuse` | Langfuse 可观测性 |
| Trace | `eino-ext/callbacks/cozeloop` | CozeLoop 可观测性 |

### VectorStore

基于 [qdrant/go-client](https://github.com/qdrant/go-client) 封装的向量存储层，位于 `component/vectorstore.go`：

| 方法 | 说明 |
|------|------|
| `EmbedQuery` | 文本转向量（预计算，用于并发检索） |
| `Store` | 向量化文档并存入 Qdrant（含自动建 collection） |
| `Retrieve` / `RetrieveContent` | 文本检索（内含 embedding） |
| `RetrieveByVector` / `RetrieveContentByVector` | 向量直接检索（跳过 embedding） |
| `Delete` | 按 ID 删除文档 |
| `List` | 列出 collection 中的文档 |


## LLM 厂商配置

| Provider   | `provider` 值 | 默认 `baseUrl` | 备注 |
|------------|--------------|----------------|------|
| OpenAI     | `openai` | 官方地址 | 直接使用 |
| DeepSeek   | `deepseek` | `https://api.deepseek.com/v1` | 自动设置 |
| Qwen       | `qwen` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | 自动设置 |
| Ollama     | `ollama` | `http://localhost:11434/v1` | 本地部署 |
| OpenRouter | `openrouter` | `https://openrouter.ai/api/v1` | 多模型网关 |

所有厂商均可通过自定义 `baseUrl` 覆盖默认地址。

## License

[MIT](LICENSE)