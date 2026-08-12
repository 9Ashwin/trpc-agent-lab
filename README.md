# trpc-agent-lab

一个基于 [trpc-agent-go](https://github.com/trpc-group/trpc-agent-go) 的 DeepSeek Agent 项目，目标是练手并吃透 trpc-agent-go 的核心能力。后端 Go 提供 AG-UI（SSE）服务，前端 React + Vite + TypeScript 实现流式聊天界面。

## 能力清单

| 能力 | 说明 | 状态 |
|------|------|------|
| Session 持久化 | 多会话 + 历史，SQLite 存储 | ✅ |
| 长期记忆 | 跨会话记住用户偏好/事实（memory_add/search/update/delete） | ✅ |
| Agent Skills | `SKILL.md` 技能，配合 codeexecutor 运行 | ✅ |
| Workspace | 隔离工作区 + 代码执行（workspace_exec） | ✅ |
| 多 Agent Team | coordinator + researcher/coder/writer 专家编排 | ✅ |
| MCP 接入 | 外部 MCP server（stdio / streamable_http / sse） | ✅ |
| Knowledge RAG | 知识库问答（需 embedding 端点，默认关闭） | ✅ |

## 架构

```
┌─────────────┐   POST /agui (AG-UI SSE)   ┌──────────────────────────────────────┐
│  React 前端  │ ─────────────────────────▶ │ Go 后端（trpc-agent-go）              │
│  Vite + TS   │ ◀───────────────────────── │  runner + session + memory + agui     │
└─────────────┘    text/event-stream 流式    │  skill + workspace + team + mcp + kb │
                                             └───────────────┬──────────────────────┘
                                                             │ HTTPS
                                                             ▼
                                             https://api.deepseek.com (deepseek-v4-*)
```

## 目录结构

```
backend/
├── cmd/server/main.go          # 入口，只装配
└── internal/
    ├── config/                 # 配置加载（.env + 环境变量）
    ├── agent/                  # agent / team 构建 + 系统指令
    ├── tools/                  # 内置函数工具（calculator / time）
    ├── skill/                  # （trpc-agent-go 内置）
    ├── mcp/                    # 外部 MCP server 加载
    ├── knowledge/              # RAG 知识库（可选）
    └── server/                 # runner + session + memory + agui 装配
├── skills/                     # Agent Skills（SKILL.md）
│   ├── file-tools/
│   └── python-math/
├── mcp_servers.json            # 外部 MCP 配置
└── go.mod                      # replace 指向本地 trpc-agent-go

frontend/                       # React + Vite + TS，AG-UI SSE 客户端
```

## 快速开始

```bash
# 后端（需要本地 trpc-agent-go，见下）
cd backend
cat > .env <<EOF
DEEPSEEK_API_KEY=sk-xxx
EOF
go run ./cmd/server
# 默认 http://127.0.0.1:8080/agui

# 前端
cd frontend
npm install
npm run dev
```

`go.mod` 通过 `replace` 指向本地 `trpc-agent-go`（`../../tr-demo/trpc-agent-go`）。先 clone 到对应路径：

```bash
mkdir -p ~/workspace/tr-demo && cd ~/workspace/tr-demo
git clone https://github.com/trpc-group/trpc-agent-go.git
```

## 配置项

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `DEEPSEEK_API_KEY` | — | 必填 |
| `MODEL` | `deepseek-v4-pro` | `deepseek-v4-flash` / `deepseek-v4-pro` |
| `ADDRESS` | `127.0.0.1:8080` | 监听地址 |
| `AGUI_PATH` | `/agui` | AG-UI 路径 |
| `DATA_DIR` | `./data` | SQLite 数据目录 |
| `SKILLS_DIR` | `./skills` | skills 目录 |
| `WORKSPACE_DIR` | `./data/workspace` | 隔离工作区 |
| `TEAM_MODE` | `true` | 是否启用多 agent team |
| `MCP_SERVERS_FILE` | `./mcp_servers.json` | MCP 配置 |
| `KNOWLEDGE_ENABLED` | `false` | 启用 RAG |
| `EMBEDDER_MODEL` / `EMBEDDER_API_KEY` / `EMBEDDER_BASE_URL` | — | embedding 配置 |

## 参考

- [trpc-agent-go](https://github.com/trpc-group/trpc-agent-go)：腾讯 Go Agent 框架
- [openclaw](https://github.com/trpc-group/trpc-agent-go/tree/main/openclaw)：trpc-agent-go 生态的完整 agent 应用样板（渠道/会话/投递/skills/subagent）

## License

Apache-2.0
