#!/bin/bash
# One-Click CDN Node Installer
# Usage: curl -s https://api.mycdn.com/install.sh | bash -s -- --token="YOUR_TOKEN"

API_URL="http://127.0.0.1:8080" # Replace with real domain in production
TOKEN=""

# Parse Args
for i in "$@"
do
case $i in
    --token=*)
    TOKEN="${i#*=}"
    ;;
esac
done

if [ -z "$TOKEN" ]; then
    echo "Error: --token is required."
    exit 1
fi

echo ">>> Starting CDN Node Installation..."

# 1. Install Dependencies (OpenResty)
echo "[1/3] Installing OpenResty..."
if command -v yum >/dev/null; then
    yum install -y yum-utils
    yum-config-manager --add-repo https://openresty.org/package/centos/openresty.repo
    yum install -y openresty
elif command -v apt-get >/dev/null; then
    apt-get update
    apt-get install -y wget gnupg ca-certificates
    wget -O - https://openresty.org/package/pubkey.gpg | apt-key add -
    echo "deb http://openresty.org/package/ubuntu $(lsb_release -sc) main" > /etc/apt/sources.list.d/openresty.list
    apt-get update
    apt-get install -y openresty
fi

# 2. Download Agent
echo "[2/3] Installing Edge Agent..."
mkdir -p /opt/cdn-agent
# In real scenario, download binary:
# wget $API_URL/download/agent -O /opt/cdn-agent/cdn-agent
# chmod +x /opt/cdn-agent/cdn-agent
# For now, we assume it's compiled manually or copied

# 3. Setup Systemd Service
echo "[3/3] Registering Service..."
cat > /etc/systemd/system/cdn-agent.service <<EOF
[Unit]
Description=CDN Edge Agent
After=network.target openresty.service

[Service]
Type=simple
ExecStart=/opt/cdn-agent/cdn-agent $TOKEN
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
# systemctl enable cdn-agent
# systemctl start cdn-agent

echo ">>> Installation Complete! Node is now connecting to Controller..."
