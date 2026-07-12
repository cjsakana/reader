# CLAUDE.md

本文件为 Claude Code（claude.ai/code）在此仓库中工作时提供指引。

## 命令

### 后端 (Go)

```bash
# 运行服务（仓库根目录）
go run main.go

# 运行全部测试
go test ./...

# 运行单个包的测试
go test ./config/...
go test ./milvus/...

# 编译
go build -o reader.exe .
```

### 前端 (Vue 3)

```bash
cd frontend

# 安装依赖
npm install

# 开发服务器（代理到 :8080）
npm run dev

# 生产构建 → frontend/dist/
npm run build
```

Gin 服务端托管预构建的 `frontend/dist/` 作为 SPA 回退。本地开发时需同时运行 Go 服务端（`go run main.go`）和 Vite 开发服务器（`npm run dev`）。Vite 开发服务器会将 `/api` 请求代理至 `:8080`。

## 架构

中文小说阅读器，支持 AI 知识提取。Go 1.24 + Gin 后端，Vue 3 + Element Plus 前端。

### 小说导入的请求生命周期

1. 上传 `.txt` → `handlers/book.go` → `services/book.go:ImportBook`
2. `pkg/encoding` 检测 BOM/UTF-8/GBK/GB18030 编码；`pkg/chapter` 通过正则匹配常见中文章节标题（第X章、第X节等）进行章节分割
3. 书籍及章节通过 GORM 写入 SQLite（`repository/book.go`）

### 基于进度的异步解析（防剧透）

`PUT /api/books/:id/progress` 更新读者位置后，向 `services/parse_manager.go` 发送信号。`ParseManager`（互斥锁保护，基于 goroutine）仅调度当前进度及之前章节的后台解析任务，确保未读章节不会被处理。

每个章节的解析（`services/knowledge.go:ParseChapter`）包含以下步骤：
1. `pkg/chunker` 将文本分割为不超过 700 字符的重叠块（80 字符重叠，优先按段落边界划分）
2. 通过 `llm/client.go` 依次调用三次 LLM，以结构化 JSON 分别提取角色及别名、关系、事件
3. `services/merge.go` 对结果进行去重，写入 SQLite，包含频次追踪及章节范围生命周期

### RAG 对话流水线

`services/chat.go` 中的 `POST /api/books/:id/ask`：
- 从 Milvus 向量搜索中获取相关文本块（当前已禁用 — `main.go` 中 `milvusClient` 为 `nil`，代码路径有 nil 保护）
- 回退至从 SQLite 中查询当前章节及之前的角色/关系/事件
- 将最近 6 条对话消息作为历史上下文
- 调用 LLM，系统提示词强制要求"无剧透、不使用外部知识"

### LLM + 向量数据库

- LLM 集成基于 Cloudwego 的 **Eino** 框架（`llm/client.go`）：提供 `ChatCompletion` 和 `CreateEmbeddings`
- 向量数据库为 Milvus v2（`milvus/client.go`）：HNSW 索引，余弦相似度，维度 1536 — **当前已禁用**；如需启用，可在 `main.go` 中取消 Milvus 初始化代码块的注释
- 启动时从 `config.yaml` 加载配置（`config/config.go`）

### 响应封装

所有 API 响应均使用 `pkg/response` 定义的格式：`{code: 0, msg: "ok", data: ...}`。错误码 1001–2003 定义于 `pkg/response/response.go`。前端 `src/api/index.js` 负责解包该封装格式。

### 前端状态

三个 Pinia store（`src/stores/`）：`reader`（当前书籍/章节）、`knowledge`（角色/关系/事件）、`chat`（对话历史）。侧边栏知识面板（`CharacterPanel`、基于 D3 的 `RelationshipGraph`、`EventTimeline`）均按读者当前章节位置在前端进行筛选。

## 关键配置

`config.yaml` 控制 LLM 端点/模型、Milvus 连接、服务端口（`8080`）、SQLite 路径（`./data/reader.db`）以及书籍目录（`./data/books`）。`pkg/envutil` 包提供环境变量覆盖及回退默认值，但主要配置方式为 YAML 文件。
