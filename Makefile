.PHONY: test test-unit test-integration run serve start stop restart build build-server build-all clean coverage lint fmt check deps tidy verify web-dev web-build

# 默认目标
.DEFAULT_GOAL := help

# Go 命令
GO := go

# 测试标志
TEST_FLAGS := -v -race

## help: 显示帮助信息
help:
	@echo "可用的命令:"
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# @/ {print $$2}' | sort

## test: 运行所有测试（跳过集成测试）
test:
	@echo "运行单元测试..."
	@$(GO) test $(TEST_FLAGS) -short ./...

## test-unit: 运行单元测试
.PHONY: test-unit
test-unit:
	@echo "运行单元测试..."
	@$(GO) test $(TEST_FLAGS) -short ./...

## test-integration: 运行集成测试（需要 API Key）
.PHONY: test-integration
test-integration:
	@echo "运行集成测试..."
	@$(GO) test $(TEST_FLAGS) ./... -run TestRunIntegrationTests

## test-verbose: 运行详细测试
.PHONY: test-verbose
test-verbose:
	@echo "运行详细测试..."
	@$(GO) test $(TEST_FLAGS) -v ./...

## run: 运行命令行主程序
.PHONY: run
run:
	@echo "运行命令行主程序..."
	@$(GO) run ./cmd/agent

## serve: 启动 Web 聊天服务（默认 :8080）
.PHONY: serve
serve:
	@echo "启动 Web 聊天服务 http://localhost:8080 ..."
	@$(GO) run ./cmd/server

## build: 构建命令行程序
.PHONY: build
build:
	@echo "构建命令行程序..."
	@if not exist bin mkdir bin
	@$(GO) build -o bin/trpc_agent.exe ./cmd/agent

## build-server: 构建 Web 服务程序
.PHONY: build-server
build-server:
	@echo "构建 Web 服务程序..."
	@if not exist bin mkdir bin
	@$(GO) build -o bin/trpc_agent_server.exe ./cmd/server

## build-all: 构建全部程序
.PHONY: build-all
build-all: web-build build build-server

## start: 构建并在后台启动 Web 服务
.PHONY: start
start:
	@scripts\start.bat

## stop: 停止运行中的 Web 服务
.PHONY: stop
stop:
	@scripts\stop.bat

## restart: 重启 Web 服务
.PHONY: restart
restart:
	@scripts\restart.bat

## clean: 清理构建文件
clean:
	@echo "清理构建文件..."
	@$(GO) clean
	@rm -rf bin
	@rm -f coverage.out coverage.html

## coverage: 生成测试覆盖率报告
.PHONY: coverage
coverage:
	@echo "生成测试覆盖率报告..."
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

## lint: 代码检查
.PHONY: lint
lint:
	@echo "运行代码检查..."
	@$(GO) vet ./...

## fmt: 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@$(GO) fmt ./...

## check: 检查代码格式
.PHONY: check
check:
	@echo "检查代码格式..."
	@$(GO) fmt ./...
	@git diff --exit-code

## deps: 下载依赖
.PHONY: deps
deps:
	@echo "下载依赖..."
	@$(GO) mod download

## tidy: 整理依赖
.PHONY: tidy
tidy:
	@echo "整理依赖..."
	@$(GO) mod tidy

## verify: 验证依赖
.PHONY: verify
verify:
	@echo "验证依赖..."
	@$(GO) mod verify

## web-dev: 启动前端开发服务器（热更新，API 代理到 :8080）
.PHONY: web-dev
web-dev:
	@cd web && npm run dev

## web-build: 构建前端到 Go embed 目录
.PHONY: web-build
web-build:
	@echo "构建前端..."
	@cd web && npm run build
