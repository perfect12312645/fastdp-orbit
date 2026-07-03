.PHONY: build build-frontend build-linux-amd64 build-linux-arm64 package clean

APP_NAME := orbit
DEPLOY_DIR := ./deploy
RELEASE_DIR := ./releases
BACKEND_DIR := ./backend
FRONTEND_DIR := ./frontend

# 版本信息
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# 产物目录（按版本隔离）
RELEASE_AMD64 := $(RELEASE_DIR)/$(VERSION)/linux-amd64
RELEASE_ARM64 := $(RELEASE_DIR)/$(VERSION)/linux-arm64

# ldflags: 注入版本信息 + 去掉调试信息 (-w -s)
LDFLAGS := -ldflags "-s -w \
	-X 'fastdp-orbit/backend/pkg/version.Version=$(VERSION)' \
	-X 'fastdp-orbit/backend/pkg/version.GitCommit=$(GIT_COMMIT)' \
	-X 'fastdp-orbit/backend/pkg/version.BuildDate=$(BUILD_DATE)'"

# ==================== 全部构建 ====================

build: build-frontend build-linux-amd64 build-linux-arm64 package
	@echo "===== 构建完成: $(VERSION) ====="

# ==================== 前端构建 ====================

build-frontend:
	@echo "构建前端..."
	@cd $(FRONTEND_DIR) && npm run build
	@echo "前端构建完成"

# ==================== Go 交叉编译 ====================

build-linux-amd64:
	@echo "编译 linux/amd64..."
	@mkdir -p $(RELEASE_AMD64)
	@cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../$(RELEASE_AMD64)/orbit-server ./cmd/server
	@cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../$(RELEASE_AMD64)/orbit-agent ./cmd/agent
	@cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../$(RELEASE_AMD64)/orbitctl ./cmd/cli
	@echo "linux/amd64 编译完成"

build-linux-arm64:
	@echo "编译 linux/arm64..."
	@mkdir -p $(RELEASE_ARM64)
	@cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o ../$(RELEASE_ARM64)/orbit-server ./cmd/server
	@cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o ../$(RELEASE_ARM64)/orbit-agent ./cmd/agent
	@cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o ../$(RELEASE_ARM64)/orbitctl ./cmd/cli
	@echo "linux/arm64 编译完成"

# ==================== 发布包打包 ====================

package:
	@echo "打包发布文件..."
	# 前端产物
	@mkdir -p $(RELEASE_AMD64)/dist $(RELEASE_ARM64)/dist
	@cp -r $(FRONTEND_DIR)/dist/* $(RELEASE_AMD64)/dist/
	@cp -r $(FRONTEND_DIR)/dist/* $(RELEASE_ARM64)/dist/
	# 部署文件（configs、脚本、service、k8s）
	@cp -r $(DEPLOY_DIR)/* $(RELEASE_AMD64)/
	@cp -r $(DEPLOY_DIR)/* $(RELEASE_ARM64)/
	# 赋予脚本可执行权限
	@chmod +x $(RELEASE_AMD64)/*.sh 2>/dev/null || true
	@chmod +x $(RELEASE_ARM64)/*.sh 2>/dev/null || true
	@echo "发布包已打包到 $(RELEASE_DIR)/$(VERSION)/"

# ==================== 清理 ====================

clean:
	@echo "清理..."
	@rm -rf $(RELEASE_DIR)
	@echo "清理完成"
