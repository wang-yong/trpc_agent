// Package calculator 提供基础的四则运算能力，可作为 Agent 的函数工具使用。
package calculator

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// Input 计算器工具的输入参数。
type Input struct {
	Operation string  `json:"operation" description:"运算类型：add, subtract, multiply, divide"`
	A         float64 `json:"a" description:"第一个操作数"`
	B         float64 `json:"b" description:"第二个操作数"`
}

// Output 计算器工具的输出。
type Output struct {
	Result float64 `json:"result" description:"计算结果"`
}

// Calculate 执行加减乘除运算。
func Calculate(ctx context.Context, input Input) (Output, error) {
	var result float64
	switch input.Operation {
	case "add":
		result = input.A + input.B
	case "subtract":
		result = input.A - input.B
	case "multiply":
		result = input.A * input.B
	case "divide":
		if input.B == 0 {
			return Output{}, fmt.Errorf("除数不能为零")
		}
		result = input.A / input.B
	default:
		return Output{}, fmt.Errorf("不支持的运算类型: %s", input.Operation)
	}
	return Output{Result: result}, nil
}

// NewCalculatorTool 创建计算器函数工具。
func NewCalculatorTool() tool.Tool {
	return function.NewFunctionTool(
		Calculate,
		function.WithName("calculator"),
		function.WithDescription("执行加减乘除运算"),
	)
}
