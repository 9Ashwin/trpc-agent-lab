// Package config loads runtime configuration from environment variables and a
// local .env file. Command-line flags (in cmd/server) override these values.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime settings for the server.
type Config struct {
	// Model is the DeepSeek model name (deepseek-v4-flash / deepseek-v4-pro).
	Model string
	// APIKey is the DeepSeek API key.
	APIKey string
	// Address is the AG-UI HTTP listen address.
	Address string
	// AGUIPath is the AG-UI endpoint path.
	AGUIPath string
	// DataDir is where SQLite databases (session/memory) live.
	DataDir string
	// SkillsDir is the directory of Agent Skills (SKILL.md files).
	SkillsDir string
	// WorkspaceDir is the isolated workspace the agent can read/write/run code in.
	WorkspaceDir string
	// TeamMode enables the coordinator multi-agent team (default true).
	TeamMode bool
	// MCPServersFile is the JSON config of external MCP servers (optional).
	MCPServersFile string
	// KnowledgeEnabled enables the RAG knowledge base (needs an embedder).
	KnowledgeEnabled bool
	// EmbedderModel / EmbedderAPIKey / EmbedderBaseURL configure the embedder.
	EmbedderModel   string
	EmbedderAPIKey  string
	EmbedderBaseURL string
	// KnowledgeDir is the directory of documents to index.
	KnowledgeDir string
}

// Load reads configuration from .env (if present) and environment variables,
// applying defaults. It returns an error when required values are missing.
func Load() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		Model:            envOr("MODEL", "deepseek-v4-pro"),
		APIKey:           os.Getenv("DEEPSEEK_API_KEY"),
		Address:          envOr("ADDRESS", "127.0.0.1:8080"),
		AGUIPath:         envOr("AGUI_PATH", "/agui"),
		DataDir:          envOr("DATA_DIR", "./data"),
		SkillsDir:        envOr("SKILLS_DIR", "./skills"),
		WorkspaceDir:     envOr("WORKSPACE_DIR", "./data/workspace"),
		TeamMode:         envBool("TEAM_MODE", true),
		MCPServersFile:   envOr("MCP_SERVERS_FILE", "./mcp_servers.json"),
		KnowledgeEnabled: envBool("KNOWLEDGE_ENABLED", false),
		EmbedderModel:    envOr("EMBEDDER_MODEL", ""),
		EmbedderAPIKey:   envOr("EMBEDDER_API_KEY", ""),
		EmbedderBaseURL:  envOr("EMBEDDER_BASE_URL", ""),
		KnowledgeDir:     envOr("KNOWLEDGE_DIR", "./knowledge"),
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("未设置 DEEPSEEK_API_KEY：请在 backend/.env 填写，或 export DEEPSEEK_API_KEY=sk-xxx")
	}
	return cfg, nil
}

// envOr returns the value of env var key, falling back to fallback when empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envBool returns the boolean value of env var key, falling back when unset.
func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return fallback
}

// loadDotEnv parses a minimal KEY=VALUE .env file. It never overrides an
// already-set environment variable.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}
