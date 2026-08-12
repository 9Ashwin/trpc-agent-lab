# Pro-Me · DeepSeek Agent

一个基于 [trpc-agent-go](https://github.com/trpc-group/trpc-agent-go) 的 Agent 项目，后端用 Go 提供
AG-UI（SSE）服务，前端用 React + Vite + TypeScript 实现流式聊天界面，模型使用 DeepSeek。

## 架构

```
┌─────────────┐   POST /agui (AG-UI SSE)   ┌──────────────────────────────┐
│   React 前端 │ ─────────────────────────▶ │ Go 后端（trpc-agent-go）      │
│  Vite + TS   │ ◀───────────────────────── │  llmagent + runner + agui    │
└─────────────┘    text/event-stream 流式    │  openai.VariantDeepSeek      │
                                             └───────────────┬──────────────┘
                                                             │ HTTPS
                                                             ▼
                                             https://api.deepseek.com (deepseek-v4-flash)
```

- **后端**：`trpc-agent-go` 的 `llmagent` + `runner`，通过 `server/agui` 暴露 AG-UI 协议的 SSE 端点。
- **模型**：DeepSeek（`deepseek-v4-flash` / `deepseek-v4-pro`），通过 `openai.VariantDeepSeek` 接入。
- **工具**：内置 `calculator`（四则/幂运算）与 `get_current_time`（时区时间）两个函数工具，演示 Agent 的工具调用能力。
- **前端**：React + TypeScript，手写 AG-UI SSE 客户端，流式渲染正文、思考过程与工具调用。

## 目录结构

```
pro-me/
├── backend/            # Go 后端
│   ├── go.mod          # 通过 replace 指向本地 trpc-agent-go
│   ├── main.go         # 模型/工具/Agent/AG-UI 服务
│   ├── .env            # 本地配置（含 API Key，勿提交）
│   └── .env.example
├── frontend/           # React 前端
│   ├── package.json
│   ├── vite.config.ts  # /agui 代理到后端
│   └── src/
│       ├── App.tsx     # 聊天 UI
│       └── agui/sse.ts # SSE 客户端
└── README.md
```

## 环境要求

- Go ≥ 1.24
- Node.js ≥ 18（建议 20+）
- 本机已检出 `trpc-agent-go`（后端通过 `replace` 引用本地路径 `../../tr-demo/trpc-agent-go`）

## 快速开始

### 1. 配置 API Key

编辑 `backend/.env`（或直接 export 环境变量）：

```bash
DEEPSEEK_API_KEY=sk-xxx
MODEL=deepseek-v4-flash   # 可选：deepseek-v4-pro 更强推理
```

> `.env` 已被 `.gitignore` 忽略，提交时请使用 `.env.example` 作为模板。

### 2. 启动后端

```bash
cd backend
go mod tidy   # 首次运行，生成 go.sum
go run .
```

看到日志即成功：

```
AG-UI: 服务已就绪，模型=deepseek-v4-flash，地址 http://127.0.0.1:8080/agui
```

### 3. 启动前端

另开一个终端：

```bash
cd frontend
npm install
npm run dev
```

浏览器打开 http://localhost:5173 ，即可与 Agent 对话。试试：

- `帮我算一下 123 + 456 等于多少`（触发 calculator 工具）
- `纽约现在是几点？`（触发 get_current_time 工具）

## 配置项

| 环境变量 / 启动参数 | 默认值 | 说明 |
| --- | --- | --- |
| `DEEPSEEK_API_KEY` | — | DeepSeek API Key（必填） |
| `MODEL` / `-model` | `deepseek-v4-flash` | 模型名，可选 `deepseek-v4-pro` |
| `ADDRESS` / `-address` | `127.0.0.1:8080` | 后端监听地址 |
| `AGUI_PATH` / `-path` | `/agui` | AG-UI 端点路径 |

后端通过命令行参数 `-model`、`-address`、`-path` 可覆盖 `.env` 中的值（环境变量优先级高于默认值）。

## 说明

- **协议选择**：本项目走框架原生的 **Chat Completions**（`/chat/completions`，OpenAI 兼容）。DeepSeek
  新推出的 **Responses API**（`/responses`）是无状态接口，主要面向 Codex；而 trpc-agent-go 的模型层基于
  Chat Completions，会话状态由框架的 runner/session 管理。两者 `base_url` 均为 `https://api.deepseek.com`，
  API Key 通用。
- **模型命名**：`deepseek-chat` / `deepseek-reasoner` 已于 2026-07-24 停用，请使用 `deepseek-v4-flash` 或
  `deepseek-v4-pro`。
- **思考模式**：`VariantDeepSeek` 已内置 DeepSeek 的思考格式处理；前端对 `REASONING_*` 事件做了兼容渲染。
