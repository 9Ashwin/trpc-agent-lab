// Package main 启动一个基于 trpc-agent-go 的 AG-UI 服务端，
// 模型使用 DeepSeek（deepseek-v4-flash / deepseek-v4-pro），
// 通过 SSE 向 React 前端流式推送对话与工具调用事件。
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/log"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/server/agui"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

func main() {
	// 启动前加载 backend/.env（若存在），方便本地填写 API Key 而无需手动 export。
	loadDotEnv(".env")

	modelName := flag.String("model", envOr("MODEL", "deepseek-v4-flash"), "DeepSeek 模型名：deepseek-v4-flash / deepseek-v4-pro")
	address := flag.String("address", envOr("ADDRESS", "127.0.0.1:8080"), "监听地址")
	path := flag.String("path", envOr("AGUI_PATH", "/agui"), "AG-UI HTTP 路径")
	flag.Parse()

	apiKey := envOr("DEEPSEEK_API_KEY", "")
	if apiKey == "" {
		log.Fatalf("未设置 DEEPSEEK_API_KEY。请在 backend/.env 中填写，或执行 export DEEPSEEK_API_KEY=sk-xxx")
	}

	// DeepSeek 使用 OpenAI 兼容协议；WithVariant(VariantDeepSeek) 会自动带上
	// base_url（https://api.deepseek.com）与 DeepSeek 的思考模式格式。
	modelInstance := openai.New(*modelName,
		openai.WithVariant(openai.VariantDeepSeek),
		openai.WithAPIKey(apiKey),
	)

	generationConfig := model.GenerationConfig{
		MaxTokens:   intPtr(2048),
		Temperature: floatPtr(0.7),
		Stream:      true,
	}

	calculatorTool := function.NewFunctionTool(
		calculator,
		function.WithName("calculator"),
		function.WithDescription("四则运算与幂运算。operation 支持 add/subtract/multiply/divide/power，a 与 b 为参与运算的两个数字。"),
	)
	timeTool := function.NewFunctionTool(
		getCurrentTime,
		function.WithName("get_current_time"),
		function.WithDescription("获取当前时间。timezone 为 IANA 时区名（如 Asia/Shanghai、America/New_York），留空则返回服务器本地时间。"),
	)

	agent := llmagent.New(
		"deepseek-agent",
		llmagent.WithModel(modelInstance),
		llmagent.WithGenerationConfig(generationConfig),
		llmagent.WithTools([]tool.Tool{calculatorTool, timeTool}),
		llmagent.WithInstruction("你是一个乐于助人的中文助手。遇到需要计算或获取当前时间的问题时，请使用提供的工具。"),
	)

	r := runner.NewRunner(agent.Info().Name, agent)
	defer r.Close()

	server, err := agui.New(r, agui.WithPath(*path))
	if err != nil {
		log.Fatalf("创建 AG-UI 服务失败: %v", err)
	}

	log.Infof("AG-UI: 服务已就绪，模型=%s，地址 http://%s%s", *modelName, *address, *path)
	if err := http.ListenAndServe(*address, server.Handler()); err != nil {
		log.Fatalf("服务停止: %v", err)
	}
}

// calculator 实现基础运算工具。
func calculator(_ context.Context, args calculatorArgs) (calculatorResult, error) {
	var result float64
	switch args.Operation {
	case "add", "+":
		result = args.A + args.B
	case "subtract", "-":
		result = args.A - args.B
	case "multiply", "*":
		result = args.A * args.B
	case "divide", "/":
		if args.B == 0 {
			return calculatorResult{}, fmt.Errorf("除数不能为 0")
		}
		result = args.A / args.B
	case "power", "^":
		result = math.Pow(args.A, args.B)
	default:
		return calculatorResult{}, fmt.Errorf("不支持的运算: %s", args.Operation)
	}
	return calculatorResult{Result: result}, nil
}

type calculatorArgs struct {
	Operation string  `json:"operation" description:"add/subtract/multiply/divide/power"`
	A         float64 `json:"a" description:"第一个数字"`
	B         float64 `json:"b" description:"第二个数字"`
}

type calculatorResult struct {
	Result float64 `json:"result"`
}

// getCurrentTime 返回指定时区的当前时间。
func getCurrentTime(_ context.Context, args currentTimeArgs) (currentTimeResult, error) {
	now := time.Now()
	loc := time.Local
	if strings.TrimSpace(args.Timezone) != "" {
		l, err := time.LoadLocation(args.Timezone)
		if err != nil {
			return currentTimeResult{}, fmt.Errorf("无效时区 %q: %w", args.Timezone, err)
		}
		loc = l
	}
	return currentTimeResult{
		Timezone: loc.String(),
		Time:     now.In(loc).Format(time.RFC3339),
	}, nil
}

type currentTimeArgs struct {
	Timezone string `json:"timezone" description:"IANA 时区名，如 Asia/Shanghai，留空为服务器本地时区"`
}

type currentTimeResult struct {
	Timezone string `json:"timezone"`
	Time     string `json:"time"`
}

func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }

// envOr 返回环境变量 key 的值，若为空则返回 fallback。
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv 极简 .env 加载：仅解析 KEY=VALUE 行，且不覆盖已存在的环境变量。
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
		if key == "" {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}
