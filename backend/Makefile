.PHONY: build build-server build-agent build-cli build-linux-amd64 build-frontend build-all clean test lint proto docker-build static

# 应用名称
APP_NAME := orbit
BUILD_DIR := ./build
STATIC_DIR := ./static
DIST_DIR := ./dist
FRONTEND_DIR := ../frontend

# 版本信息 - 编译时自动获取
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# ldflags - 注入版本信息到 pkg/version
LDFLAGS := -ldflags "\
	-X 'fastdp-orbit/backend/pkg/version.Version=$(VERSION)' \
	-X 'fastdp-orbit/backend/pkg/version.GitCommit=$(GIT_COMMIT)' \
	-X 'fastdp-orbit/backend/pkg/version.BuildDate=$(BUILD_DATE)'"

# 编译全部
build: build-server build-agent build-cli

# 编译 Linux AMD64 版本
build-linux-amd64:
	@echo "Building linux-amd64 binaries $(VERSION)..."
	@mkdir -p $(BUILD_DIR)/linux-amd64
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/orbit-server ./cmd/server
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/orbit-agent ./cmd/agent
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/orbitctl ./cmd/cli
	@echo "Linux AMD64 binaries ready in $(BUILD_DIR)/linux-amd64/"

# 编译前端
build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && npm run build
	@echo "Frontend ready in $(DIST_DIR)/"

# 编译全部（Linux AMD64 + 前端 + 静态文件）
build-all: proto build-linux-amd64 build-frontend static
	@echo "Build all completed!"

# 编译 Web Server
build-server:
	@echo "Building orbit-server $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/orbit-server ./cmd/server

# 编译 Agent
build-agent:
	@echo "Building orbit-agent $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/orbit-agent ./cmd/agent

# 编译 CLI
build-cli:
	@echo "Building orbitctl $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/orbitctl ./cmd/cli

# 运行 Server
run-server:
	@go run ./cmd/server

# 运行 Agent
run-agent:
	@go run ./cmd/agent

# 运行 CLI
run-cli:
	@go run ./cmd/cli

# 清理
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)

# 准备静态文件（用于分发）
static: build-linux-amd64
	@echo "Preparing static files..."
	@mkdir -p $(STATIC_DIR)
	@cp $(BUILD_DIR)/linux-amd64/orbit-server $(STATIC_DIR)/
	@cp $(BUILD_DIR)/linux-amd64/orbit-agent $(STATIC_DIR)/
	@cp $(BUILD_DIR)/linux-amd64/orbitctl $(STATIC_DIR)/
	@cp configs/server.toml $(STATIC_DIR)/
	@cp configs/agent.toml $(STATIC_DIR)/
	@cp ../deploy/install-server.sh $(STATIC_DIR)/
	@cp ../deploy/install-agent.sh $(STATIC_DIR)/
	@cp ../deploy/orbit-server.service $(STATIC_DIR)/
	@cp ../deploy/orbit-agent.service $(STATIC_DIR)/
	@if [ -d "$(DIST_DIR)" ]; then \
		mkdir -p $(STATIC_DIR)/dist; \
		cp -r $(DIST_DIR)/* $(STATIC_DIR)/dist/; \
		echo "Frontend dist copied to $(STATIC_DIR)/dist/"; \
	fi
	@chmod +x $(STATIC_DIR)/*.sh 2>/dev/null || true
	@echo "Static files ready in $(STATIC_DIR)/"

# 测试
test:
	@echo "Running tests..."
	@go test ./...

# 代码检查
lint:
	@echo "Running linter..."
	@golangci-lint run

# 生成 proto 文件
proto:
	@echo "Generating proto files..."
	@protoc --go_out=. --go_opt=module=fastdp-orbit/backend --go-grpc_out=. --go-grpc_opt=module=fastdp-orbit/backend proto/agent/agent.proto
	@protoc --go_out=. --go_opt=module=fastdp-orbit/backend --go-grpc_out=. --go-grpc_opt=module=fastdp-orbit/backend proto/server/server.proto

# Docker 构建
docker-build:
	@echo "Building docker images..."
	@docker build -t orbit-server:$(VERSION) -f Dockerfile.server .
	@docker build -t orbit-agent:$(VERSION) -f Dockerfile.agent .
