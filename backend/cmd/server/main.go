// Command server is the single entry point for trpc-agent-lab. It only loads
// configuration and delegates to the server assembly layer.
package main

import (
	"flag"
	"log"

	"github.com/9Ashwin/trpc-agent-lab/backend/internal/config"
	"github.com/9Ashwin/trpc-agent-lab/backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// CLI flags override config/env values.
	flag.StringVar(&cfg.Model, "model", cfg.Model, "DeepSeek 模型名：deepseek-v4-flash / deepseek-v4-pro")
	flag.StringVar(&cfg.Address, "address", cfg.Address, "监听地址")
	flag.StringVar(&cfg.AGUIPath, "path", cfg.AGUIPath, "AG-UI HTTP 路径")
	flag.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "SQLite 数据目录")
	flag.Parse()

	if err := server.Run(cfg); err != nil {
		log.Fatalf("server: %v", err)
	}
}
