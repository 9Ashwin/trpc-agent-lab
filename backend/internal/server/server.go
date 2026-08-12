// Package server wires the runner, session service, memory service, and the
// AG-UI HTTP endpoint together. This is the assembly layer; it contains no
// tool/agent business logic.
package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"trpc.group/trpc-go/trpc-agent-go/agent"
	localexec "trpc.group/trpc-go/trpc-agent-go/codeexecutor/local"
	"trpc.group/trpc-go/trpc-agent-go/memory/sqlite"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/server/agui"
	sessionsqlite "trpc.group/trpc-go/trpc-agent-go/session/sqlite"
	"trpc.group/trpc-go/trpc-agent-go/skill"

	"trpc.group/trpc-go/trpc-agent-go/log"

	agentpkg "github.com/9Ashwin/trpc-agent-lab/backend/internal/agent"
	"github.com/9Ashwin/trpc-agent-lab/backend/internal/config"
	knowledgesrv "github.com/9Ashwin/trpc-agent-lab/backend/internal/knowledge"
	mcpsrv "github.com/9Ashwin/trpc-agent-lab/backend/internal/mcp"
	"github.com/9Ashwin/trpc-agent-lab/backend/internal/tools"
)

const appName = "trpc-agent-lab"

// Run assembles services and starts the AG-UI HTTP server, blocking until it
// exits.
func Run(cfg *config.Config) error {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Session persistence (SQLite).
	sessionDB, err := sql.Open("sqlite3", "file:"+filepath.Join(cfg.DataDir, "sessions.db")+"?_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open session db: %w", err)
	}
	defer sessionDB.Close()
	sessionService, err := sessionsqlite.NewService(sessionDB,
		sessionsqlite.WithSessionEventLimit(500),
	)
	if err != nil {
		return fmt.Errorf("init session service: %w", err)
	}

	// Long-term memory (SQLite).
	memoryDB, err := sql.Open("sqlite3", "file:"+filepath.Join(cfg.DataDir, "memory.db")+"?_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open memory db: %w", err)
	}
	defer memoryDB.Close()
	memoryService, err := sqlite.NewService(memoryDB)
	if err != nil {
		return fmt.Errorf("init memory service: %w", err)
	}

	// Agent tools = built-in utilities + memory tools.
	allTools := append(tools.All(), memoryService.Tools()...)

	// Isolated workspace (code executor).
	exec := localexec.New(
		localexec.WithWorkDir(cfg.WorkspaceDir),
		localexec.WithWorkspaceMode(localexec.WorkspaceModeTrustedLocal),
	)

	// Agent Skills repository.
	repo, err := skill.NewFSRepository(cfg.SkillsDir)
	if err != nil {
		return fmt.Errorf("init skills repo: %w", err)
	}

	// External MCP tool sets (optional; missing config → none).
	mcpToolSets, err := mcpsrv.LoadToolSets(cfg.MCPServersFile)
	if err != nil {
		return fmt.Errorf("load mcp servers: %w", err)
	}

	// RAG knowledge base (optional; disabled without an embedder).
	kb, err := knowledgesrv.Build(cfg)
	if err != nil {
		return fmt.Errorf("build knowledge: %w", err)
	}

	deps := agentpkg.Deps{
		ExtraTools: allTools,
		ToolSets:   mcpToolSets,
		Repo:       repo,
		Exec:       exec,
		Knowledge:  kb,
	}

	// Build the top-level agent: coordinator team (default) or single agent.
	var llmAgent agent.Agent
	if cfg.TeamMode {
		llmAgent = agentpkg.BuildTeam(cfg, deps)
	} else {
		llmAgent = agentpkg.Build(cfg, deps)
	}

	r := runner.NewRunner(appName, llmAgent,
		runner.WithSessionService(sessionService),
		runner.WithMemoryService(memoryService),
	)
	defer r.Close()

	aguiServer, err := agui.New(r, agui.WithPath(cfg.AGUIPath))
	if err != nil {
		return fmt.Errorf("create agui server: %w", err)
	}

	log.Infof("trpc-agent-lab: 服务就绪，模型=%s，地址 http://%s%s", cfg.Model, cfg.Address, cfg.AGUIPath)
	return http.ListenAndServe(cfg.Address, aguiServer.Handler())
}
