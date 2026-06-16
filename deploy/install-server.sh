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
if [ $# -lt 1 ]; then
    echo -e "${RED}错误：请传入 server IP 作为第一个参数！${NC}"
    echo "用法：$0 <server_ip> [server_port]"
    exit 1
fi

# ==============================================
# 配置参数
# ==============================================
SERVER_IP="${1}"
SERVER_PORT="${2:-8080}"
INSTALL_DIR="/opt/fastdp-orbit"
CONFIG_DIR="/etc/fastdp-orbit"
CERTS_DIR="${CONFIG_DIR}/certs"
SYSTEMD_DIR="/etc/systemd/system"
CONFIG_FILE="${CONFIG_DIR}/server.toml"
SERVER_BIN_NAME="orbit-server"
SERVER_BIN_PATH="${INSTALL_DIR}/${SERVER_BIN_NAME}"
SYSTEMD_SERVICE_NAME="orbit-server.service"
SYSTEMD_SERVICE_PATH="${SYSTEMD_DIR}/${SYSTEMD_SERVICE_NAME}"

# ==============================================
# 验证IP地址
# ==============================================
echo -e "${YELLOW}===== 验证IP地址 =====${NC}"

# 验证是否为IPv4格式
if ! echo "${SERVER_IP}" | grep -qE '^([0-9]{1,3}\.){3}[0-9]{1,3}$'; then
    echo -e "${RED}错误：${SERVER_IP} 不是有效的IPv4地址${NC}"
    exit 1
fi

# 验证不能是0.0.0.0
if [ "${SERVER_IP}" = "0.0.0.0" ]; then
    echo -e "${RED}错误：IP地址不能是 0.0.0.0${NC}"
    exit 1
fi

# 验证不能是回环地址
if echo "${SERVER_IP}" | grep -qE '^127\.'; then
    echo -e "${RED}错误：IP地址不能是回环地址 (127.x.x.x)${NC}"
    exit 1
fi

# 验证是否为本机IP
if ! ip addr | grep -q "inet ${SERVER_IP}"; then
    echo -e "${RED}错误：${SERVER_IP} 不是本机IP地址${NC}"
    echo "本机IP地址列表："
    ip addr | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '^127\.'
    exit 1
fi

echo -e "${GREEN}IP验证通过: ${SERVER_IP}${NC}"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Orbit Server 安装脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo "Server IP: ${SERVER_IP}"
echo "Server Port: ${SERVER_PORT}"
echo ""

# ==============================================
# 步骤1：检查当前目录文件
# ==============================================
echo -e "${YELLOW}===== 1. 检查当前目录文件 =====${NC}"

if [ ! -f "./${SERVER_BIN_NAME}" ]; then
    echo -e "${RED}错误：当前目录未找到 ${SERVER_BIN_NAME}${NC}"
    echo "请确保以下文件在当前目录："
    echo "  - ${SERVER_BIN_NAME}"
    echo "  - server.toml"
    echo "  - ${SYSTEMD_SERVICE_NAME}"
    exit 1
fi

if [ ! -f "./server.toml" ]; then
    echo -e "${RED}错误：当前目录未找到 server.toml${NC}"
    exit 1
fi

if [ ! -f "./${SYSTEMD_SERVICE_NAME}" ]; then
    echo -e "${RED}错误：当前目录未找到 ${SYSTEMD_SERVICE_NAME}${NC}"
    exit 1
fi

echo -e "${GREEN}文件检查通过${NC}"

# ==============================================
# 步骤2：停止运行中的服务
# ==============================================
echo -e "\n${YELLOW}===== 2. 停止 orbit-server 服务 =====${NC}"
if systemctl is-active --quiet orbit-server; then
    systemctl stop orbit-server
    echo -e "${GREEN}服务已停止${NC}"
else
    echo "服务未运行，无需停止"
fi

# ==============================================
# 步骤3：清理旧文件
# ==============================================
echo -e "\n${YELLOW}===== 3. 清理旧文件 =====${NC}"
rm -f "${SERVER_BIN_PATH}"
rm -f "${CONFIG_FILE}"
rm -f "${SYSTEMD_SERVICE_PATH}"
echo -e "${GREEN}旧文件已清理${NC}"

# ==============================================
# 步骤4：创建目录
# ==============================================
echo -e "\n${YELLOW}===== 4. 创建目录 =====${NC}"
mkdir -p "${INSTALL_DIR}"
mkdir -p "${CONFIG_DIR}"
mkdir -p "${CERTS_DIR}/server"
echo -e "${GREEN}目录创建完成${NC}"

# ==============================================
# 步骤5：复制文件到系统目录
# ==============================================
echo -e "\n${YELLOW}===== 5. 复制文件 =====${NC}"

# 复制二进制文件
cp "./${SERVER_BIN_NAME}" "${SERVER_BIN_PATH}"
chmod +x "${SERVER_BIN_PATH}"
echo "已复制: ${SERVER_BIN_NAME} -> ${SERVER_BIN_PATH}"

# 复制配置文件
cp "./server.toml" "${CONFIG_FILE}"
echo "已复制: server.toml -> ${CONFIG_FILE}"

# 复制systemd服务文件
cp "./${SYSTEMD_SERVICE_NAME}" "${SYSTEMD_SERVICE_PATH}"
echo "已复制: ${SYSTEMD_SERVICE_NAME} -> ${SYSTEMD_SERVICE_PATH}"

# 复制orbitctl到/usr/local/bin
if [ -f "./orbitctl" ]; then
    cp "./orbitctl" "/usr/local/bin/orbitctl"
    chmod +x "/usr/local/bin/orbitctl"
    echo "已复制: orbitctl -> /usr/local/bin/orbitctl"
else
    echo -e "${YELLOW}警告：当前目录未找到 orbitctl，跳过${NC}"
fi

echo -e "${GREEN}文件复制完成${NC}"

# ==============================================
# 步骤6：生成TLS自签证书
# ==============================================
echo -e "\n${YELLOW}===== 6. 生成TLS自签证书 =====${NC}"

# CA证书
if [ ! -f "${CERTS_DIR}/ca.crt" ]; then
    echo "生成CA证书..."
    openssl genrsa -out "${CERTS_DIR}/ca.key" 2048 2>/dev/null
    openssl req -x509 -new -nodes -key "${CERTS_DIR}/ca.key" -sha256 -days 3650 \
        -out "${CERTS_DIR}/ca.crt" -subj "/CN=Orbit CA" 2>/dev/null
    echo -e "${GREEN}CA证书生成完成${NC}"
else
    echo "CA证书已存在，跳过生成"
fi

# Server gRPC证书
if [ ! -f "${CERTS_DIR}/server/grpc.crt" ]; then
    echo "生成Server gRPC证书..."
    mkdir -p "${CERTS_DIR}/server"
    openssl genrsa -out "${CERTS_DIR}/server/grpc.key" 2048 2>/dev/null
    
    # 创建扩展文件（不绑定IP，使用通配符）
    cat > /tmp/server-ext.cnf << EOF
[req]
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_ca]
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = *.local
EOF
    
    openssl req -new -key "${CERTS_DIR}/server/grpc.key" \
        -out "${CERTS_DIR}/server/grpc.csr" \
        -subj "/CN=Orbit Server" 2>/dev/null
    openssl x509 -req -in "${CERTS_DIR}/server/grpc.csr" \
        -CA "${CERTS_DIR}/ca.crt" -CAkey "${CERTS_DIR}/ca.key" -CAcreateserial \
        -out "${CERTS_DIR}/server/grpc.crt" -days 3650 -sha256 \
        -extfile /tmp/server-ext.cnf -extensions v3_ca 2>/dev/null
    rm -f /tmp/server-ext.cnf
    echo -e "${GREEN}Server gRPC证书生成完成${NC}"
else
    echo "Server gRPC证书已存在，跳过生成"
fi

# Server HTTP证书（包含IP SANs，供orbitctl验证）
if [ ! -f "${CERTS_DIR}/server/http.crt" ]; then
    echo "生成Server HTTP证书（包含IP: ${SERVER_IP}）..."
    openssl genrsa -out "${CERTS_DIR}/server/http.key" 2048 2>/dev/null
    
    cat > /tmp/http-ext.cnf << EOF
[req]
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_ca]
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = *.local
IP.1 = ${SERVER_IP}
EOF
    
    openssl req -new -key "${CERTS_DIR}/server/http.key" \
        -out "${CERTS_DIR}/server/http.csr" \
        -subj "/CN=Orbit Server HTTP" 2>/dev/null
    openssl x509 -req -in "${CERTS_DIR}/server/http.csr" \
        -CA "${CERTS_DIR}/ca.crt" -CAkey "${CERTS_DIR}/ca.key" -CAcreateserial \
        -out "${CERTS_DIR}/server/http.crt" -days 3650 -sha256 \
        -extfile /tmp/http-ext.cnf -extensions v3_ca 2>/dev/null
    rm -f /tmp/http-ext.cnf
    echo -e "${GREEN}Server HTTP证书生成完成（IP: ${SERVER_IP}）${NC}"
    echo -e "${YELLOW}注意：如果Server IP地址变更，需要重新生成证书！${NC}"
else
    echo "Server HTTP证书已存在，跳过生成"
fi

# 设置证书权限
chmod 600 "${CERTS_DIR}"/*.key "${CERTS_DIR}/server"/*.key 2>/dev/null || true
chmod 644 "${CERTS_DIR}"/*.crt "${CERTS_DIR}/server"/*.crt 2>/dev/null || true

# ==============================================
# 步骤7：生成Agent通用证书（供所有Agent使用）
# ==============================================
echo -e "\n${YELLOW}===== 7. 生成Agent通用证书 =====${NC}"

AGENT_CERTS_DIR="${CERTS_DIR}/agent"
mkdir -p "${AGENT_CERTS_DIR}"

if [ ! -f "${AGENT_CERTS_DIR}/grpc.crt" ]; then
    echo "生成Agent gRPC证书..."
    openssl genrsa -out "${AGENT_CERTS_DIR}/grpc.key" 2048 2>/dev/null
    
    # 创建扩展文件（Agent证书不限制IP，可被任何Agent使用）
    cat > /tmp/agent-ext.cnf << EOF
[req]
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_ca]
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = *.local
EOF
    
    openssl req -new -key "${AGENT_CERTS_DIR}/grpc.key" \
        -out "${AGENT_CERTS_DIR}/grpc.csr" \
        -subj "/CN=Orbit Agent" 2>/dev/null
    openssl x509 -req -in "${AGENT_CERTS_DIR}/grpc.csr" \
        -CA "${CERTS_DIR}/ca.crt" -CAkey "${CERTS_DIR}/ca.key" -CAcreateserial \
        -out "${AGENT_CERTS_DIR}/grpc.crt" -days 3650 -sha256 \
        -extfile /tmp/agent-ext.cnf -extensions v3_ca 2>/dev/null
    rm -f /tmp/agent-ext.cnf
    echo -e "${GREEN}Agent gRPC证书生成完成${NC}"
else
    echo "Agent gRPC证书已存在，跳过生成"
fi

# 设置Agent证书权限
chmod 600 "${AGENT_CERTS_DIR}"/*.key 2>/dev/null || true
chmod 644 "${AGENT_CERTS_DIR}"/*.crt 2>/dev/null || true

# ==============================================
# 步骤8：创建static目录（供Agent安装使用）
# ==============================================
echo -e "\n${YELLOW}===== 8. 创建static目录 =====${NC}"

STATIC_DIR="${INSTALL_DIR}/static"
mkdir -p "${STATIC_DIR}"

# 复制Agent相关文件
if [ -f "./install-agent.sh" ]; then
    cp "./install-agent.sh" "${STATIC_DIR}/"
    chmod +x "${STATIC_DIR}/install-agent.sh"
    echo "已复制: install-agent.sh -> ${STATIC_DIR}/"
fi

if [ -f "./agent.toml" ]; then
    cp "./agent.toml" "${STATIC_DIR}/"
    echo "已复制: agent.toml -> ${STATIC_DIR}/"
fi

if [ -f "./orbit-agent" ]; then
    cp "./orbit-agent" "${STATIC_DIR}/"
    chmod +x "${STATIC_DIR}/orbit-agent"
    echo "已复制: orbit-agent -> ${STATIC_DIR}/"
fi

if [ -f "./orbit-agent.service" ]; then
    cp "./orbit-agent.service" "${STATIC_DIR}/"
    echo "已复制: orbit-agent.service -> ${STATIC_DIR}/"
fi

rm -rf ${INSTALL_DIR}/dist
cp -a dist ${INSTALL_DIR}

# 复制CA证书到static目录
mkdir -p "${STATIC_DIR}/certs/agent"
cp "${CERTS_DIR}/ca.crt" "${STATIC_DIR}/certs"
echo "已复制: ca.crt -> ${STATIC_DIR}/certs"

# 复制Agent证书到static目录
cp "${AGENT_CERTS_DIR}/grpc.crt" "${STATIC_DIR}/certs/agent/"
cp "${AGENT_CERTS_DIR}/grpc.key" "${STATIC_DIR}/certs/agent/"
echo "已复制: agent certificates -> ${STATIC_DIR}/certs/agent/"

echo -e "${GREEN}static目录创建完成${NC}"

# ==============================================
# 步骤8：更新配置文件
# ==============================================
echo -e "\n${YELLOW}===== 8. 更新配置文件 =====${NC}"

# 更新HTTP监听地址
sed -i "s/^address = \"0.0.0.0\"/address = \"${SERVER_IP}\"/" "${CONFIG_FILE}" 2>/dev/null || \
    sed -i "0,/address = \"0.0.0.0\"/s//address = \"${SERVER_IP}\"/" "${CONFIG_FILE}"

# 更新HTTP端口
sed -i "s/^port = 8080/port = ${SERVER_PORT}/" "${CONFIG_FILE}" 2>/dev/null || \
    sed -i "0,/port = 8080/s//port = ${SERVER_PORT}/" "${CONFIG_FILE}"

# 更新gRPC地址（与HTTP地址相同）
sed -i "s/^address = \"0.0.0.0\"/address = \"${SERVER_IP}\"/" "${CONFIG_FILE}" 2>/dev/null || \
    sed -i "0,/address = \"0.0.0.0\"/s//address = \"${SERVER_IP}\"/" "${CONFIG_FILE}"

# 更新gRPC端口
sed -i "s/^port = 9090/port = 9090/" "${CONFIG_FILE}" 2>/dev/null || \
    sed -i "0,/port = 9090/s//port = 9090/" "${CONFIG_FILE}"

# 开启HTTP TLS
sed -i 's/enabled = false/enabled = true/' "${CONFIG_FILE}"

echo -e "${GREEN}配置文件更新完成${NC}"

# ==============================================
# 步骤9：启动服务
# ==============================================
echo -e "\n${YELLOW}===== 9. 启动服务 =====${NC}"
systemctl daemon-reload
systemctl enable --now orbit-server

# 等待服务启动
sleep 2

# 最终状态检查
if systemctl is-active --quiet orbit-server; then
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Orbit Server 安装完成！${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo "配置文件：${CONFIG_FILE}"
    echo "安装目录：${INSTALL_DIR}"
    echo "证书目录：${CERTS_DIR}"
    echo "static目录：${STATIC_DIR}"
    echo ""
    echo -e "${YELLOW}查看日志：${NC}"
    echo "journalctl -u orbit-server -f"
    echo ""
    echo -e "${YELLOW}⚠ 重要提示：${NC}"
    echo -e "${YELLOW}  Server HTTP证书包含IP SANs（${SERVER_IP}），如果IP地址变更，${NC}"
    echo -e "${YELLOW}  需要删除证书并重新运行脚本生成新证书：${NC}"
    echo -e "${YELLOW}  rm -f ${CERTS_DIR}/server/http.* && bash $0 <新IP> [端口]${NC}"
else
    echo -e "\n${RED}===== 错误：服务启动失败！查看日志：journalctl -u orbit-server -f =====${NC}"
    exit 1
fi
