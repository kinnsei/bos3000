#!/usr/bin/env bash
set -euo pipefail

# BOS3000 Server Installation Script
# Supports: Ubuntu 22.04/24.04, CentOS 8+, AlmaLinux 8+
#
# Usage:
#   sudo bash install.sh --version v1.0.0 --eip 47.xx.xx.xx
#   sudo bash install.sh --version v1.0.0 --eip 47.xx.xx.xx --fs-address 10.0.0.2:8021

# ---- Parse arguments ----
VERSION=""
EIP=""
FS_ADDRESS="localhost:8021"
FS_PASSWORD="ClueCon"
INSTALL_DIR="/opt/bos3000"
DB_PASSWORD=""
JWT_SECRET=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --version)    VERSION="$2"; shift 2 ;;
    --eip)        EIP="$2"; shift 2 ;;
    --fs-address) FS_ADDRESS="$2"; shift 2 ;;
    --fs-password) FS_PASSWORD="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: sudo bash install.sh --version <vX.Y.Z> --eip <elastic-ip>"
      echo ""
      echo "Options:"
      echo "  --version      Required. Version tag (e.g. v1.0.0)"
      echo "  --eip          Required. Public/Elastic IP for SIP signaling"
      echo "  --fs-address   FreeSWITCH ESL address (default: localhost:8021)"
      echo "  --fs-password  FreeSWITCH ESL password (default: ClueCon)"
      echo "  --install-dir  Installation directory (default: /opt/bos3000)"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "$VERSION" ]]; then
  echo "Error: --version is required"
  exit 1
fi
if [[ -z "$EIP" ]]; then
  echo "Error: --eip is required"
  exit 1
fi

# ---- Checks ----
if [[ $EUID -ne 0 ]]; then
  echo "Error: This script must be run as root (use sudo)"
  exit 1
fi

echo "============================================"
echo "  BOS3000 Installation  $VERSION"
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

echo "[1/9] System: $OS_ID $OS_VERSION ($(uname -m))"

IS_DEB=false
IS_RPM=false
case "$OS_ID" in
  ubuntu|debian)
    IS_DEB=true
    ;;
  centos|almalinux|rocky|rhel)
    IS_RPM=true
    ;;
  *)
    echo "Warning: Untested OS ($OS_ID). Proceeding anyway..."
    if command -v apt-get &>/dev/null; then IS_DEB=true;
    elif command -v dnf &>/dev/null; then IS_RPM=true;
    else echo "Error: No supported package manager found"; exit 1; fi
    ;;
esac

# ---- Install PostgreSQL 16 ----
echo "[2/9] Installing PostgreSQL 16..."

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
echo "[3/9] Setting up database..."

DB_PASSWORD=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)

su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='bos3000'\" | grep -q 1" 2>/dev/null || \
  su - postgres -c "psql -c \"CREATE ROLE bos3000 WITH LOGIN PASSWORD '${DB_PASSWORD}';\""

su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname='bos3000'\" | grep -q 1" 2>/dev/null || \
  su - postgres -c "psql -c \"CREATE DATABASE bos3000 OWNER bos3000;\""

echo "  Database bos3000 ready."

# ---- Create system user ----
echo "[4/9] Creating system user..."

id -u bos3000 &>/dev/null || useradd -r -m -d "$INSTALL_DIR" -s /bin/bash bos3000
mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/data"

echo "  User bos3000 ready."

# ---- Deploy binary ----
echo "[5/9] Deploying BOS3000 $VERSION..."

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Copy deployment files
cp -r "$SCRIPT_DIR"/* "$INSTALL_DIR/" 2>/dev/null || true

# If docker image exists, load it
DOCKER_IMG="$SCRIPT_DIR/../dist/bos3000-${VERSION}-docker.tar.gz"
if [[ -f "$DOCKER_IMG" ]]; then
  echo "  Loading Docker image..."
  docker load < "$DOCKER_IMG"
fi

echo "  Deployed to $INSTALL_DIR"

# ---- Generate JWT secret ----
JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)

# ---- Write environment file ----
echo "[6/9] Writing configuration..."

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

echo "  Configuration written to $INSTALL_DIR/.env"

# ---- Set Encore secrets (if encore is installed) ----
echo "[7/9] Setting up secrets..."

if command -v encore &>/dev/null; then
  echo "$JWT_SECRET" | encore secret set --type local JWTSecret 2>/dev/null || true
  echo "  Encore secrets configured."
else
  echo "  Encore CLI not found, skipping Encore secrets. Set manually if using Encore runtime."
fi

# ---- Configure firewall ----
echo "[8/9] Configuring firewall..."

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
echo "[9/9] Setting up systemd service..."

cp "$INSTALL_DIR/bos3000.service" /etc/systemd/system/bos3000.service 2>/dev/null || true
systemctl daemon-reload
systemctl enable bos3000

echo ""
echo "============================================"
echo "  BOS3000 $VERSION Installation Complete"
echo "============================================"
echo ""
echo "  Install dir:  $INSTALL_DIR"
echo "  Config file:  $INSTALL_DIR/.env"
echo "  Service:      systemctl {start|stop|restart|status} bos3000"
echo "  Logs:         journalctl -u bos3000 -f"
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
echo "  IMPORTANT:"
echo "    1. Change the default admin password immediately!"
echo "    2. Save the database password shown above."
echo "    3. Configure FreeSWITCH ESL in $INSTALL_DIR/.env"
echo "    4. Start the service: sudo systemctl start bos3000"
echo ""
echo "  Reset admin password:"
echo "    bash $INSTALL_DIR/reset-admin-password.sh"
echo ""
