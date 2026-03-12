#!/usr/bin/env bash
set -euo pipefail

# BOS3000 Server Installation Script (single-binary, no Docker)
# Supports: Ubuntu 22.04/24.04, CentOS 8+, AlmaLinux 8+
#
# Usage:
#   sudo bash install.sh --eip 47.xx.xx.xx
#   sudo bash install.sh --eip 47.xx.xx.xx --fs-address 10.0.0.2:8021

# ---- Parse arguments ----
EIP=""
FS_ADDRESS="localhost:8021"
FS_PASSWORD="ClueCon"
INSTALL_DIR="/opt/bos3000"

while [[ $# -gt 0 ]]; do
  case $1 in
    --eip)         EIP="$2"; shift 2 ;;
    --fs-address)  FS_ADDRESS="$2"; shift 2 ;;
    --fs-password) FS_PASSWORD="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: sudo bash install.sh --eip <elastic-ip> [options]"
      echo ""
      echo "Options:"
      echo "  --eip          Required. Public/Elastic IP for SIP signaling"
      echo "  --fs-address   FreeSWITCH ESL address (default: localhost:8021)"
      echo "  --fs-password  FreeSWITCH ESL password (default: ClueCon)"
      echo "  --install-dir  Installation directory (default: /opt/bos3000)"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "$EIP" ]]; then
  echo "Error: --eip is required"
  exit 1
fi

if [[ $EUID -ne 0 ]]; then
  echo "Error: This script must be run as root (use sudo)"
  exit 1
fi

# Detect version from .version file in the same directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="unknown"
if [[ -f "$SCRIPT_DIR/.version" ]]; then
  VERSION="$(cat "$SCRIPT_DIR/.version")"
fi

# Check binary exists
if [[ ! -f "$SCRIPT_DIR/bos3000" ]]; then
  echo "Error: bos3000 binary not found in $SCRIPT_DIR"
  echo "       Make sure you extracted the deployment bundle correctly."
  exit 1
fi

echo "============================================"
echo "  BOS3000 Installation  v${VERSION}"
echo "============================================"
echo ""

# ---- Detect OS ----
if [[ -f /etc/os-release ]]; then
  . /etc/os-release
  OS_ID="$ID"
  OS_VERSION="$VERSION_ID"
else
  echo "Error: Cannot detect OS. /etc/os-release not found."
  exit 1
fi

echo "[1/8] System: $OS_ID $OS_VERSION ($(uname -m))"

IS_DEB=false
IS_RPM=false
case "$OS_ID" in
  ubuntu|debian)
    IS_DEB=true ;;
  centos|almalinux|rocky|rhel)
    IS_RPM=true ;;
  *)
    echo "Warning: Untested OS ($OS_ID). Proceeding..."
    if command -v apt-get &>/dev/null; then IS_DEB=true;
    elif command -v dnf &>/dev/null; then IS_RPM=true;
    else echo "Error: No supported package manager found"; exit 1; fi
    ;;
esac

# ---- Install PostgreSQL 16 ----
echo "[2/8] Installing PostgreSQL 16..."

if $IS_DEB; then
  if ! command -v psql &>/dev/null; then
    apt-get update -qq
    apt-get install -y -qq curl gnupg lsb-release >/dev/null
    sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/pgdg.gpg
    apt-get update -qq
    apt-get install -y -qq postgresql-16 >/dev/null
  fi
  systemctl enable --now postgresql
elif $IS_RPM; then
  if ! command -v psql &>/dev/null; then
    dnf install -y -q https://download.postgresql.org/pub/repos/yum/reporpms/EL-$(rpm -E %{rhel})-x86_64/pgdg-redhat-repo-latest.noarch.rpm 2>/dev/null || true
    dnf -qy module disable postgresql 2>/dev/null || true
    dnf install -y -q postgresql16-server postgresql16 >/dev/null
    /usr/pgsql-16/bin/postgresql-16-setup initdb
  fi
  systemctl enable --now postgresql-16
fi

echo "  PostgreSQL ready."

# ---- Create database and user ----
echo "[3/8] Setting up database..."

DB_PASSWORD=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)

su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='bos3000'\" | grep -q 1" 2>/dev/null || \
  su - postgres -c "psql -c \"CREATE ROLE bos3000 WITH LOGIN PASSWORD '${DB_PASSWORD}';\""

su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname='bos3000'\" | grep -q 1" 2>/dev/null || \
  su - postgres -c "psql -c \"CREATE DATABASE bos3000 OWNER bos3000;\""

echo "  Database bos3000 ready."

# ---- Create system user & deploy binary ----
echo "[4/8] Deploying binary to $INSTALL_DIR ..."

id -u bos3000 &>/dev/null || useradd -r -m -d "$INSTALL_DIR" -s /bin/bash bos3000
mkdir -p "$INSTALL_DIR/bin"

cp "$SCRIPT_DIR/bos3000" "$INSTALL_DIR/bin/bos3000"
chmod +x "$INSTALL_DIR/bin/bos3000"

# Copy encore metadata (needed by runtime)
if [[ -f "$SCRIPT_DIR/encore-meta" ]]; then
  cp "$SCRIPT_DIR/encore-meta" "$INSTALL_DIR/bin/encore-meta"
fi

# Copy utility scripts
cp "$SCRIPT_DIR/reset-admin-password.sh" "$INSTALL_DIR/" 2>/dev/null || true
cp "$SCRIPT_DIR/.version" "$INSTALL_DIR/" 2>/dev/null || true

echo "  Binary installed: $INSTALL_DIR/bin/bos3000"

# ---- Generate secrets & write env ----
echo "[5/8] Writing configuration..."

JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)

cat > "$INSTALL_DIR/.env" << ENVEOF
# BOS3000 Production Configuration — Generated $(date -Iseconds)
# Version: $VERSION

# Database
BOS3000_DB_HOST=localhost
BOS3000_DB_PORT=5432
BOS3000_DB_USER=bos3000
BOS3000_DB_PASSWORD=${DB_PASSWORD}
BOS3000_DB_NAME=bos3000

# FreeSWITCH ESL
FS_MODE=real
FS_ADDRESS=${FS_ADDRESS}
FS_PASSWORD=${FS_PASSWORD}

# Secrets
JWT_SECRET=${JWT_SECRET}

# Network
EIP=${EIP}
SIP_EXTERNAL_IP=${EIP}
ENVEOF

chmod 600 "$INSTALL_DIR/.env"
chown -R bos3000:bos3000 "$INSTALL_DIR"

echo "  Config: $INSTALL_DIR/.env"

# ---- Configure firewall ----
echo "[6/8] Configuring firewall..."

if command -v ufw &>/dev/null; then
  ufw allow 4000/tcp comment "BOS3000 API" >/dev/null 2>&1 || true
  ufw allow 5060/udp comment "SIP Signaling" >/dev/null 2>&1 || true
  ufw allow 5060/tcp comment "SIP Signaling TCP" >/dev/null 2>&1 || true
  ufw allow 5061/tcp comment "SIP TLS" >/dev/null 2>&1 || true
  ufw allow 8021/tcp comment "FreeSWITCH ESL" >/dev/null 2>&1 || true
  ufw allow 16384:32768/udp comment "RTP Media" >/dev/null 2>&1 || true
  echo "  UFW rules added."
elif command -v firewall-cmd &>/dev/null; then
  firewall-cmd --permanent --add-port=4000/tcp >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=5060/udp >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=5060/tcp >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=5061/tcp >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=8021/tcp >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=16384-32768/udp >/dev/null 2>&1 || true
  firewall-cmd --reload >/dev/null 2>&1 || true
  echo "  firewalld rules added."
else
  echo "  No firewall detected, skipping."
fi

# ---- Install systemd service ----
echo "[7/8] Setting up systemd service..."

cat > /etc/systemd/system/bos3000.service << SVCEOF
[Unit]
Description=BOS3000 VoIP Callback Platform
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=bos3000
Group=bos3000
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/.env
ExecStart=$INSTALL_DIR/bin/bos3000
Restart=on-failure
RestartSec=5
LimitNOFILE=65535
StandardOutput=journal
StandardError=journal
SyslogIdentifier=bos3000

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable bos3000

# ---- Migrate database ----
echo "[8/8] Database migrations will run on first start."

echo ""
echo "============================================"
echo "  BOS3000 v${VERSION} Installation Complete"
echo "============================================"
echo ""
echo "  Install dir:  $INSTALL_DIR"
echo "  Binary:       $INSTALL_DIR/bin/bos3000"
echo "  Config:       $INSTALL_DIR/.env"
echo ""
echo "  Database:"
echo "    Host:       localhost:5432"
echo "    Database:   bos3000"
echo "    User:       bos3000"
echo "    Password:   $DB_PASSWORD"
echo ""
echo "  Default Admin Login:"
echo "    URL:        http://${EIP}:4000/admin/"
echo "    Email:      admin@localhost"
echo "    Password:   changeme123"
echo ""
echo "  Commands:"
echo "    sudo systemctl start bos3000        # Start"
echo "    sudo systemctl status bos3000       # Status"
echo "    journalctl -u bos3000 -f            # Logs"
echo "    bash $INSTALL_DIR/reset-admin-password.sh  # Reset password"
echo ""
echo "  IMPORTANT:"
echo "    1. Change the default admin password immediately!"
echo "    2. Save the database password above."
echo "    3. Start: sudo systemctl start bos3000"
echo ""
