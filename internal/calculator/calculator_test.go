package calculator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCalculate_Add 测试加法运算
func TestCalculate_Add(t *testing.T) {
	tests := []struct {
		name     string
		input    Input
		expected float64
	}{
		{
			name:     "正数相加",
			input:    Input{Operation: "add", A: 2, B: 3},
			expected: 5,
		},
		{
			name:     "负数相加",
			input:    Input{Operation: "add", A: -2, B: -3},
			expected: -5,
		},
		{
			name:     "正负数相加",
			input:    Input{Operation: "add", A: 5, B: -3},
			expected: 2,
		},
		{
			name:     "零和正数相加",
			input:    Input{Operation: "add", A: 0, B: 5},
			expected: 5,
		},
		{
			name:     "小数相加",
			input:    Input{Operation: "add", A: 2.5, B: 3.7},
			expected: 6.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := Calculate(context.Background(), tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, output.Result)
		})
	}
}

// TestCalculate_Subtract 测试减法运算
func TestCalculate_Subtract(t *testing.T) {
	tests := []struct {
		name     string
		input    Input
		expected float64
	}{
		{
			name:     "正数相减",
			input:    Input{Operation: "subtract", A: 5, B: 3},
			expected: 2,
		},
		{
			name:     "负数相减",
			input:    Input{Operation: "subtract", A: -5, B: -3},
			expected: -2,
		},
		{
			name:     "正减负",
			input:    Input{Operation: "subtract", A: 5, B: -3},
			expected: 8,
		},
		{
			name:     "相等数相减",
			input:    Input{Operation: "subtract", A: 5, B: 5},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := Calculate(context.Background(), tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, output.Result)
		})
	}
}

// TestCalculate_Multiply 测试乘法运算
func TestCalculate_Multiply(t *testing.T) {
	tests := []struct {
		name     string
		input    Input
		expected float64
	}{
		{
			name:     "正数相乘",
			input:    Input{Operation: "multiply", A: 3, B: 4},
			expected: 12,
		},
		{
			name:     "负数相乘",
			input:    Input{Operation: "multiply", A: -3, B: -4},
			expected: 12,
		},
		{
			name:     "正乘负",
			input:    Input{Operation: "multiply", A: 3, B: -4},
			expected: -12,
		},
		{
			name:     "乘以零",
			input:    Input{Operation: "multiply", A: 5, B: 0},
			expected: 0,
		},
		{
			name:     "小数相乘",
			input:    Input{Operation: "multiply", A: 2.5, B: 4},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := Calculate(context.Background(), tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, output.Result)
		})
	}
}

// TestCalculate_Divide 测试除法运算
func TestCalculate_Divide(t *testing.T) {
	tests := []struct {
		name     string
		input    Input
		expected float64
	}{
		{
			name:     "正数相除",
			input:    Input{Operation: "divide", A: 10, B: 2},
			expected: 5,
		},
		{
			name:     "负数相除",
			input:    Input{Operation: "divide", A: -10, B: -2},
			expected: 5,
		},
		{
			name:     "正除负",
			input:    Input{Operation: "divide", A: 10, B: -2},
			expected: -5,
		},
		{
			name:     "小数相除",
			input:    Input{Operation: "divide", A: 10, B: 4},
			expected: 2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := Calculate(context.Background(), tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, output.Result)
		})
	}
}

// TestCalculate_DivideByZero 测试除以零的错误处理
func TestCalculate_DivideByZero(t *testing.T) {
	input := Input{Operation: "divide", A: 10, B: 0}
	_, err := Calculate(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "除数不能为零")
}

// TestCalculate_InvalidOperation 测试无效运算类型
func TestCalculate_InvalidOperation(t *testing.T) {
	input := Input{Operation: "invalid", A: 10, B: 5}
	_, err := Calculate(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的运算类型")
}

// TestCalculate_AllOperations 测试所有运算的综合场景
func TestCalculate_AllOperations(t *testing.T) {
	operations := []struct {
		name     string
		input    Input
		expected float64
	}{
		{"加法", Input{"add", 10, 5}, 15},
		{"减法", Input{"subtract", 10, 5}, 5},
		{"乘法", Input{"multiply", 10, 5}, 50},
		{"除法", Input{"divide", 10, 5}, 2},
	}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			output, err := Calculate(context.Background(), op.input)
			assert.NoError(t, err)
			assert.Equal(t, op.expected, output.Result)
		})
	}
}
