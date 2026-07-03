# FastDP Orbit

**多机可视化运维编排平台**

基于 Agent-gRPC 的多机器运维操作编排系统，类 Ansible 的可视化替代方案。通过 Web 界面拖拽式编排运维任务，批量在远程机器上执行 Shell、文件管理、镜像管理等操作，内置 Kubernetes 离线部署方案库。

## 架构

```
┌─────────────────────────────────────────────────┐
│                    Web UI (Vue 3)                │
│  工作流画布 / 机器管理 / 方案库 / 存储管理        │
└──────────────────────┬──────────────────────────┘
                       │ HTTP (JSON API)
┌──────────────────────▼──────────────────────────┐
│              Orbit Server (Go/Gin)               │
│  认证 / 工作流引擎 / 方案库 / 阶段模板 / 文件存储  │
│  SQLite (单机) / MySQL (集群)                    │
└──────┬────────────────────────────────┬──────────┘
       │ gRPC                            │ gRPC
┌──────▼──────┐                ┌─────────▼────────┐
│  Agent 1    │    ......      │    Agent N        │
│ (Go)        │                │ (Go)              │
│ Shell/文件/ │                │ Shell/文件/       │
│ 镜像管理     │                │ 镜像管理           │
└─────────────┘                └──────────────────┘
```

- **Server**: 中心控制节点，提供 HTTP API 和 gRPC 服务，管理机器、工作流、方案库
- **Agent**: 部署在目标机器上，通过 gRPC 接收任务并执行 Shell / 文件管理 / 镜像操作 / 解压等模块
- **CLI** (`orbitctl`): 命令行工具，用于登录 Server、管理机器、获取 Agent 安装命令等

## 核心功能

| 功能 | 说明 |
|---|---|
| 可视化工作流编排 | 拖拽式构建运维流水线，支持多阶段、并行执行、条件钩子 |
| 批量机器管理 | Agent 自动注册、心跳保活、硬件信息采集 |
| 方案库 (Solution Library) | 预置/自定义运维方案，一键应用到集群 |
| 阶段模板 | 可复用的任务模板，支持版本管理和回滚 |
| 全局变量 | 跨工作流共享变量 |
| 文件存储 | 分块上传、断点续传，支持 wget 下载 |
| Kubernetes 部署 | 内置离线部署 K8s v1.32 二进制高可用集群方案 |
| CLI 管理 | orbitctl 命令行管理机器、获取 Agent 安装命令 |
| JWT 认证 | 登录鉴权，密码强度校验，首次登录强制改密 |

## 快速开始

### 1. 启动 Server

```bash
# 编译
make build-linux-amd64

# 部署到服务器
scp -r releases/v1.0.0/linux-amd64/* root@your-server:/opt/orbit/
cd /opt/orbit

# 启动
./orbit-server -c configs/server.toml
```

默认监听 `0.0.0.0:8080`（HTTP），首次运行自动创建 SQLite 数据库，初始用户: `admin / admin123`。

### 2. 部署 Agent（通过 Web 页面或 CLI）

在 Web 页面 机器管理 → 安装 Agent，或通过 CLI：

```bash
# 设置 Server 地址
orbitctl config set-server <server-ip>:8080

# 登录
orbitctl login admin

# 获取安装命令
orbitctl install
```

### 3. 访问 Web 界面

浏览器打开 `http://<server-ip>:8080`，登录后即可使用工作流编排等功能。

## 命令行工具

```bash
orbitctl config set-server <host:port>   # 设置 Server 地址
orbitctl config get-server               # 查看 Server 地址
orbitctl config path                     # 查看配置文件路径
orbitctl login <username>                # 登录
orbitctl logout                          # 退出登录
orbitctl install                         # 获取 Agent 安装命令
orbitctl machine list                    # 列出已注册机器
orbitctl machine remove <ip:port>        # 删除机器
orbitctl reset-password <username>       # 重置用户密码（需 SSH + 配置文件）
orbitctl version                         # 查看版本
```

CLI 配置文件保存在 `~/.fastdp-orbit/config.toml`。

## 构建

### 前置要求

- Go ≥ 1.26.4
- Node.js ≥ 18
- 前端依赖: `cd frontend && npm install`

### 全量构建

```bash
make build
```

产物输出到 `releases/<version>/`，包含两个架构：

```
releases/v1.0.0/
├── linux-amd64/
│   ├── orbit-server    (Server)
│   ├── orbit-agent     (Agent)
│   ├── orbitctl        (CLI)
│   ├── dist/            (前端页面)
│   ├── configs/         (配置文件)
│   ├── install-*.sh     (安装脚本)
│   ├── *.service        (systemd 服务)
│   └── k8s-*.yaml       (K8s 部署方案)
└── linux-arm64/
    └── ...
```

也可分步构建：

```bash
make build-frontend        # 编译前端
make build-linux-amd64     # 编译 linux/amd64
make build-linux-arm64     # 编译 linux/arm64
make package               # 打包发布目录
make clean                 # 清理
```

### 本地开发

```bash
cd backend && go run ./cmd/server    # 启动 Server
cd backend && go run ./cmd/agent     # 启动 Agent
cd frontend && npm run dev           # 启动前端开发服务器
```

## 目录结构

```
├── Makefile                 # 构建入口
├── deploy/                  # 部署源文件
│   ├── configs/             # 配置文件
│   ├── install-*.sh         # 安装脚本
│   ├── *.service            # systemd 服务单元
│   └── k8s-*.yaml           # K8s 部署方案
├── releases/                # 构建产物（按版本）
├── backend/
│   ├── agent/               # Agent 端代码
│   │   ├── grpc/            # gRPC 服务端
│   │   ├── handler/         # Agent RPC 处理器
│   │   └── modules/         # 任务模块（shell/file/image/unarchive）
│   ├── api/                 # HTTP API
│   │   ├── middleware/      # JWT 认证、CORS
│   │   ├── views/           # 接口处理器
│   │   └── router.go        # 路由定义
│   ├── cli/                 # CLI 命令行工具
│   │   ├── commands/        # 子命令实现
│   │   └── cliutil/         # HTTP 客户端等工具
│   ├── config/              # 配置加载
│   ├── database/            # 数据库初始化与迁移
│   ├── engine/              # 工作流执行引擎
│   ├── models/              # 数据模型
│   ├── pkg/                 # 通用工具
│   ├── proto/               # Protobuf 定义
│   ├── server/              # 服务端组件（缓存、gRPC 池等）
│   └── services/            # 业务逻辑层
├── frontend/
│   ├── src/                 # Vue 3 源码
│   │   ├── pages/           # 页面组件
│   │   ├── components/      # 通用组件
│   │   ├── stores/          # Pinia 状态管理
│   │   ├── api/             # API 调用
│   │   ├── router/          # 路由
│   │   └── utils/           # 工具函数
│   └── package.json
└── proto/                   # Protobuf 源文件
```

## Agent 模块

在远程机器上执行的原子操作单元，类似 Ansible Module：

| 模块 | 功能 |
|---|---|
| `shell` | 执行 Shell 命令/脚本 |
| `file` | 文件/目录创建、删除、权限修改 |
| `image` | Docker 镜像拉取、推送、加载、移除 |
| `unarchive` | 解压 tar.gz 等归档文件 |

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.26, Gin, GORM, Viper, Cobra |
| 前端 | Vue 3, TypeScript, Element Plus, Pinia, Vite |
| 通信 | HTTP (REST API) + gRPC (Agent) |
| 认证 | JWT (6h 过期) |
| 数据库 | SQLite (默认) / MySQL (可选) |
| 协议 | Protobuf (Agent/Server 通信) |
| 构建 | Makefile, CGO_ENABLED=0 静态编译 |

## License

MIT
