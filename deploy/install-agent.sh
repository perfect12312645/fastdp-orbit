#!/bin/bash
set -eu

# 颜色常量
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
NC="\033[0m"

# 检查root权限
if [ "$(id -u)" -ne 0 ]; then
    echo -e "${RED}错误：请以root权限执行脚本（使用sudo）${NC}"
    exit 1
fi

# 参数检查
if [ $# -lt 3 ]; then
    echo -e "${RED}错误：参数不足！${NC}"
    echo "用法：$0 <server_ip> <server_port> <token>"
    exit 1
fi

# ==============================================
# 配置参数
# ==============================================
SERVER_IP="${1}"
SERVER_PORT="${2}"
TOKEN="${3}"
SERVER_BASE_URL="https://${SERVER_IP}:${SERVER_PORT}/static"
INSTALL_DIR="/opt/fastdp-orbit"
CONFIG_DIR="/etc/fastdp-orbit"
CERTS_DIR="${CONFIG_DIR}/certs"
SYSTEMD_DIR="/etc/systemd/system"
CONFIG_FILE="${CONFIG_DIR}/agent.toml"
AGENT_BIN_NAME="orbit-agent"
AGENT_BIN_PATH="${INSTALL_DIR}/${AGENT_BIN_NAME}"
SYSTEMD_SERVICE_NAME="orbit-agent.service"
SYSTEMD_SERVICE_PATH="${SYSTEMD_DIR}/${SYSTEMD_SERVICE_NAME}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Orbit Agent 安装脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo "Server: ${SERVER_IP}:${SERVER_PORT}"
echo "Token: ${TOKEN:0:8}..."
echo ""

# 设置统一时区
timedatectl set-timezone Asia/Shanghai

# ==============================================
# 步骤1：停止运行中的服务
# ==============================================
echo -e "${YELLOW}===== 1. 停止 orbit-agent 服务 =====${NC}"
if systemctl is-active --quiet orbit-agent; then
    systemctl stop orbit-agent
    echo -e "${GREEN}服务已停止${NC}"
else
    echo "服务未运行，无需停止"
fi

# ==============================================
# 步骤2：删除旧文件
# ==============================================
echo -e "\n${YELLOW}===== 2. 清理旧文件 =====${NC}"
rm -f "${AGENT_BIN_PATH}"
rm -f "${CONFIG_FILE}"
rm -f "${SYSTEMD_SERVICE_PATH}"
echo -e "${GREEN}旧文件已清理${NC}"

# ==============================================
# 步骤3：创建目录
# ==============================================
echo -e "\n${YELLOW}===== 3. 创建目录 =====${NC}"
mkdir -p "${INSTALL_DIR}"
mkdir -p "${CONFIG_DIR}"
mkdir -p "${CERTS_DIR}/agent"
echo -e "${GREEN}目录创建完成${NC}"

# ==============================================
# 步骤4：下载文件
# ==============================================
echo -e "\n${YELLOW}===== 4. 下载安装文件 =====${NC}"

# 下载agent二进制
echo "下载 ${AGENT_BIN_NAME}..."
if ! curl -kfSL "${SERVER_BASE_URL}/${AGENT_BIN_NAME}" -o "${AGENT_BIN_PATH}"; then
    echo -e "${RED}错误：下载 ${AGENT_BIN_NAME} 失败${NC}"
    exit 1
fi
chmod +x "${AGENT_BIN_PATH}"

# 下载配置文件模板
echo "下载配置文件模板..."
if ! curl -kfSL "${SERVER_BASE_URL}/agent.toml" -o "${CONFIG_FILE}"; then
    echo -e "${RED}错误：下载配置文件失败${NC}"
    exit 1
fi

# 下载systemd服务文件
echo "下载systemd服务文件..."
if ! curl -kfSL "${SERVER_BASE_URL}/orbit-agent.service" -o "${SYSTEMD_SERVICE_PATH}"; then
    echo -e "${RED}错误：下载systemd服务文件失败${NC}"
    exit 1
fi

# 下载CA证书
echo "下载CA证书..."
if ! curl -kfSL "${SERVER_BASE_URL}/certs/ca.crt" -o "${CERTS_DIR}/ca.crt"; then
    echo -e "${RED}错误：下载CA证书失败${NC}"
    exit 1
fi

echo -e "${GREEN}文件下载完成${NC}"

# ==============================================
# 步骤5：下载Agent gRPC证书
# ==============================================
echo -e "\n${YELLOW}===== 5. 下载Agent gRPC证书 =====${NC}"

# 创建Agent证书目录
mkdir -p "${CERTS_DIR}/agent"

# 下载Agent证书
echo "下载Agent gRPC证书..."
if ! curl -kfSL "${SERVER_BASE_URL}/certs/agent/grpc.crt" -o "${CERTS_DIR}/agent/grpc.crt"; then
    echo -e "${RED}错误：下载Agent gRPC证书失败${NC}"
    exit 1
fi

echo "下载Agent gRPC私钥..."
if ! curl -kfSL "${SERVER_BASE_URL}/certs/agent/grpc.key" -o "${CERTS_DIR}/agent/grpc.key"; then
    echo -e "${RED}错误：下载Agent gRPC私钥失败${NC}"
    exit 1
fi

# 设置证书权限
chmod 600 "${CERTS_DIR}/agent"/*.key 2>/dev/null || true
chmod 644 "${CERTS_DIR}/agent"/*.crt 2>/dev/null || true

echo -e "${GREEN}Agent gRPC证书下载完成${NC}"

# ==============================================
# 步骤6：获取本机IP并更新配置文件
# ==============================================
echo -e "\n${YELLOW}===== 6. 配置Agent =====${NC}"

# 获取本机IP（优先同网段）
SERVER_PREFIX=$(echo "${SERVER_IP}" | cut -d '.' -f 1-3)
AGENT_IP=$(ip addr | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '^127\.' | grep "^${SERVER_PREFIX}\." | head -n 1)

if [ -z "${AGENT_IP}" ]; then
    AGENT_IP=$(ip addr | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '^127\.' | head -n 1)
fi

if [ -z "${AGENT_IP}" ]; then
    echo -e "${RED}警告：自动获取本机IP失败${NC}"
    echo ""
    # 仅配置已知字段
    sed -i "s/^rpcserver_host = \".*\"/rpcserver_host = \"${SERVER_IP}\"/" "${CONFIG_FILE}"
    sed -i "s/^token = \".*\"/token = \"${TOKEN}\"/" "${CONFIG_FILE}"
    echo ""
    echo -e "${RED}请手动修改配置文件中的 host 字段后启动服务：${NC}"
    echo ""
    echo "  配置文件: ${CONFIG_FILE}"
    echo "  修改字段: host = \"<your-ip>\""
    echo ""
    echo "  启动服务:"
    echo "    systemctl daemon-reload"
    echo "    systemctl enable --now orbit-agent"
    echo ""
    echo "  查看日志:"
    echo "    journalctl -u orbit-agent -f"
    exit 1
fi

echo "本机IP: ${AGENT_IP}"

# 更新配置文件
sed -i "s/^host = \".*\"/host = \"${AGENT_IP}\"/" "${CONFIG_FILE}"
sed -i "s/^rpcserver_host = \".*\"/rpcserver_host = \"${SERVER_IP}\"/" "${CONFIG_FILE}"
sed -i "s/^token = \".*\"/token = \"${TOKEN}\"/" "${CONFIG_FILE}"

echo -e "${GREEN}配置文件更新完成${NC}"

# ==============================================
# 步骤7：启动服务
# ==============================================
echo -e "\n${YELLOW}===== 7. 启动服务 =====${NC}"
systemctl daemon-reload
systemctl enable --now orbit-agent

# 等待服务启动
sleep 1

# 最终状态检查
if systemctl is-active --quiet orbit-agent; then
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Orbit Agent 安装完成！${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo "配置文件：${CONFIG_FILE}"
    echo "安装目录：${INSTALL_DIR}"
    echo "本机IP：${AGENT_IP}"
    echo ""
    echo -e "${YELLOW}查看日志：${NC}"
    echo "journalctl -u orbit-agent -f"
else
    echo -e "\n${RED}===== 错误：服务启动失败！查看日志：journalctl -u orbit-agent -f =====${NC}"
    exit 1
fi
