// Package tools collects the built-in function tools exposed to the agent.
//
// Each tool is a plain function wrapped with function.NewFunctionTool so the
// agent can call it. Tools here are generic utilities (calculation, time,
// web fetch); workspace and skill capabilities are wired separately.
package tools

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// All returns the built-in utility tools.
func All() []tool.Tool {
	return []tool.Tool{calculatorTool(), currentTimeTool()}
}

func calculatorTool() tool.Tool {
	return function.NewFunctionTool(
		calculator,
		function.WithName("calculator"),
		function.WithDescription("四则运算与幂运算。operation 支持 add/subtract/multiply/divide/power，a 与 b 为参与运算的两个数字。"),
	)
}

func currentTimeTool() tool.Tool {
	return function.NewFunctionTool(
		getCurrentTime,
		function.WithName("get_current_time"),
		function.WithDescription("获取当前时间。timezone 为 IANA 时区名（如 Asia/Shanghai），留空则返回服务器本地时间。"),
	)
}

// calculator implements basic arithmetic.
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

// getCurrentTime returns the current time in the requested timezone.
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
