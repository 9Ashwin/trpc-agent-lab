// Package mcp loads external MCP servers from a JSON config file and exposes
// them as tool sets for the agent. A missing config file disables MCP quietly;
// an unreachable server is skipped with a warning rather than failing startup.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	mcpclient "trpc.group/trpc-go/trpc-agent-go/tool/mcp"
)

// ServerConfig describes one external MCP server.
type ServerConfig struct {
	// Transport is "stdio", "streamable_http", or "sse".
	Transport string `json:"transport"`
	// ServerURL is used for streamable_http / sse transports.
	ServerURL string `json:"server_url,omitempty"`
	// Headers are optional HTTP headers.
	Headers map[string]string `json:"headers,omitempty"`
	// Command and Args are used for the stdio transport.
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	// Description is a capability summary exposed to the model.
	Description string `json:"description,omitempty"`
	// TimeoutSec is the connection timeout in seconds (default 10).
	TimeoutSec int `json:"timeout_sec,omitempty"`
}

// LoadToolSets reads the MCP config file at path and returns initialized tool
// sets. It returns (nil, nil) when the file does not exist.
func LoadToolSets(path string) ([]tool.ToolSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read mcp config: %w", err)
	}

	var servers []ServerConfig
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("parse mcp config: %w", err)
	}

	var sets []tool.ToolSet
	for _, s := range servers {
		timeout := time.Duration(s.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Second
		}

		ts := mcpclient.NewMCPToolSet(mcpclient.ConnectionConfig{
			Transport:   s.Transport,
			ServerURL:   s.ServerURL,
			Headers:     s.Headers,
			Command:     s.Command,
			Args:        s.Args,
			Description: s.Description,
			Timeout:     timeout,
		})

		if err := ts.Init(context.Background()); err != nil {
			name := s.ServerURL
			if name == "" {
				name = s.Command
			}
			fmt.Fprintf(os.Stderr, "WARN: MCP server %q 连接失败，已跳过: %v\n", name, err)
			continue
		}
		sets = append(sets, ts)
	}
	return sets, nil
}
