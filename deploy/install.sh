#!/usr/bin/env bash
set -euo pipefail

# BOS3000 Server Installation Script (single-binary, no Docker)
# Installs: PostgreSQL 16 + Redis 7 + FreeSWITCH 1.10 + BOS3000 binary
# Supports: Ubuntu 22.04/24.04, Debian 12, CentOS 8+, AlmaLinux 8+
#
# Usage:
#   sudo bash install.sh --eip 47.xx.xx.xx
#   sudo bash install.sh --eip 47.xx.xx.xx --fs-password MySecretPass

# ============================================================
#  Parse arguments
# ============================================================
EIP=""
FS_PASSWORD="ClueCon"
INSTALL_DIR="/opt/bos3000"
SKIP_FS=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --eip)         EIP="$2"; shift 2 ;;
    --fs-password) FS_PASSWORD="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --skip-freeswitch) SKIP_FS=true; shift ;;
    -h|--help)
      cat <<HELPEOF
Usage: sudo bash install.sh --eip <elastic-ip> [options]

Options:
  --eip              Required. Public/Elastic IP for SIP signaling
  --fs-password      FreeSWITCH ESL password (default: ClueCon)
  --install-dir      Installation directory (default: /opt/bos3000)
  --skip-freeswitch  Skip FreeSWITCH installation (if external FS)
HELPEOF
      exit 0
      ;;
    *) echo "Error: Unknown option: $1"; exit 1 ;;
  esac
done

[[ -z "$EIP" ]] && { echo "Error: --eip is required"; exit 1; }
[[ $EUID -ne 0 ]] && { echo "Error: Run as root (sudo)"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="unknown"
[[ -f "$SCRIPT_DIR/.version" ]] && VERSION="$(cat "$SCRIPT_DIR/.version")"

[[ ! -f "$SCRIPT_DIR/bos3000" ]] && {
  echo "Error: bos3000 binary not found in $SCRIPT_DIR"
  exit 1
}

echo "============================================"
echo "  BOS3000 Installation  v${VERSION}"
echo "============================================"
echo ""

# ============================================================
#  [1/10] Detect OS
# ============================================================
if [[ -f /etc/os-release ]]; then
  . /etc/os-release
  OS_ID="$ID"
  OS_VER="$VERSION_ID"
else
  echo "Error: /etc/os-release not found"; exit 1
fi

IS_DEB=false; IS_RPM=false
case "$OS_ID" in
  ubuntu|debian) IS_DEB=true ;;
  centos|almalinux|rocky|rhel) IS_RPM=true ;;
  *)
    if command -v apt-get &>/dev/null; then IS_DEB=true;
    elif command -v dnf &>/dev/null; then IS_RPM=true;
    else echo "Error: Unsupported OS"; exit 1; fi
    ;;
esac

ARCH=$(uname -m)
echo "[1/10] System: $OS_ID $OS_VER ($ARCH)"
[[ "$ARCH" != "x86_64" ]] && echo "  WARNING: Binary is amd64, current arch is $ARCH"

# ============================================================
#  [2/10] Install PostgreSQL 16
# ============================================================
echo "[2/10] PostgreSQL 16..."

if command -v psql &>/dev/null; then
  PG_VER=$(psql --version | grep -oP '\d+' | head -1)
  echo "  Already installed (v$PG_VER)"
else
  if $IS_DEB; then
    apt-get update -qq
    apt-get install -y -qq curl gnupg lsb-release >/dev/null
    sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/pgdg.gpg 2>/dev/null
    apt-get update -qq
    apt-get install -y -qq postgresql-16 >/dev/null
  elif $IS_RPM; then
    dnf install -y -q https://download.postgresql.org/pub/repos/yum/reporpms/EL-$(rpm -E %{rhel})-x86_64/pgdg-redhat-repo-latest.noarch.rpm 2>/dev/null || true
    dnf -qy module disable postgresql 2>/dev/null || true
    dnf install -y -q postgresql16-server postgresql16 >/dev/null
    /usr/pgsql-16/bin/postgresql-16-setup initdb 2>/dev/null || true
  fi
  echo "  Installed."
fi

# Enable and start
if $IS_RPM; then
  systemctl enable --now postgresql-16 2>/dev/null || systemctl enable --now postgresql 2>/dev/null
else
  systemctl enable --now postgresql
fi

# Configure pg_hba.conf to allow password auth for local connections
PG_HBA=$(su - postgres -c "psql -tc \"SHOW hba_file;\"" 2>/dev/null | xargs)
if [[ -n "$PG_HBA" && -f "$PG_HBA" ]]; then
  # Add md5 auth for bos3000 user before the default rules
  if ! grep -q "bos3000" "$PG_HBA" 2>/dev/null; then
    # Insert at top of file (after comments)
    sed -i '/^# TYPE/a local   all   bos3000   md5\nhost    all   bos3000   127.0.0.1/32   md5\nhost    all   bos3000   ::1/128        md5' "$PG_HBA"
    # Reload PostgreSQL to pick up changes
    systemctl reload postgresql 2>/dev/null || systemctl reload postgresql-16 2>/dev/null || true
    echo "  Configured pg_hba.conf for password auth."
  fi
fi

# ============================================================
#  [3/10] Create database
# ============================================================
echo "[3/10] Database setup..."

DB_PASSWORD=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)

# Create role (or update password if exists)
su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='bos3000'\" | grep -q 1" 2>/dev/null && {
  su - postgres -c "psql -c \"ALTER ROLE bos3000 WITH PASSWORD '${DB_PASSWORD}';\"" 2>/dev/null
  echo "  Role bos3000 exists, password updated."
} || {
  su - postgres -c "psql -c \"CREATE ROLE bos3000 WITH LOGIN PASSWORD '${DB_PASSWORD}';\"" 2>/dev/null
  echo "  Role bos3000 created."
}

# Create databases for each service
for DB_NAME in auth billing callback compliance routing webhook; do
  su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'\" | grep -q 1" 2>/dev/null || {
    su - postgres -c "psql -c \"CREATE DATABASE ${DB_NAME} OWNER bos3000;\"" 2>/dev/null
    echo "  Created database: $DB_NAME"
  }
done

# Verify connectivity
if PGPASSWORD="$DB_PASSWORD" psql -h 127.0.0.1 -U bos3000 -d auth -c "SELECT 1" &>/dev/null; then
  echo "  Database connection verified."
else
  echo "  WARNING: Cannot connect with password auth. Check pg_hba.conf manually."
fi

# ============================================================
#  [4/10] Install Redis 7
# ============================================================
echo "[4/10] Redis..."

if command -v redis-server &>/dev/null; then
  REDIS_VER=$(redis-server --version | grep -oP 'v=\K[\d.]+')
  echo "  Already installed (v$REDIS_VER)"
else
  if $IS_DEB; then
    apt-get install -y -qq redis-server >/dev/null
  elif $IS_RPM; then
    dnf install -y -q redis >/dev/null
  fi
  echo "  Installed."
fi

systemctl enable --now redis-server 2>/dev/null || systemctl enable --now redis 2>/dev/null

# Verify Redis
if redis-cli ping 2>/dev/null | grep -q PONG; then
  echo "  Redis connection verified."
else
  echo "  WARNING: Redis not responding to PING."
fi

# ============================================================
#  [5/10] Install FreeSWITCH
# ============================================================
echo "[5/10] FreeSWITCH..."

if $SKIP_FS; then
  echo "  Skipped (--skip-freeswitch)."
else
  if command -v freeswitch &>/dev/null || command -v fs_cli &>/dev/null; then
    echo "  Already installed."
  else
    if $IS_DEB; then
      # SignalWire public packages
      apt-get install -y -qq gnupg2 wget lsb-release >/dev/null
      wget -qO - https://freeswitch.signalwire.com/repo/deb/debian-release/signalwire-freeswitch-repo.gpg | gpg --dearmor -o /usr/share/keyrings/freeswitch.gpg 2>/dev/null || true
      echo "deb [signed-by=/usr/share/keyrings/freeswitch.gpg] https://freeswitch.signalwire.com/repo/deb/debian-release/ $(lsb_release -sc) main" \
        > /etc/apt/sources.list.d/freeswitch.list
      apt-get update -qq 2>/dev/null
      apt-get install -y -qq freeswitch-meta-all 2>/dev/null || {
        echo "  WARNING: SignalWire repo may require auth token."
        echo "  Install FreeSWITCH manually or use --skip-freeswitch"
      }
    elif $IS_RPM; then
      dnf install -y -q https://freeswitch.signalwire.com/repo/yum/centos-release/freeswitch-release-repo-0-1.noarch.rpm 2>/dev/null || true
      dnf install -y -q freeswitch-config-vanilla freeswitch-lang-en freeswitch-sounds-en-us-calista 2>/dev/null || {
        echo "  WARNING: FreeSWITCH install failed. Install manually."
      }
    fi
  fi

  # ---- Configure FreeSWITCH ESL ----
  FS_CONF_DIR=""
  for d in /etc/freeswitch /usr/local/freeswitch/conf /opt/freeswitch/conf; do
    [[ -d "$d" ]] && { FS_CONF_DIR="$d"; break; }
  done

  if [[ -n "$FS_CONF_DIR" ]]; then
    echo "  FS config dir: $FS_CONF_DIR"

    # Configure Event Socket (ESL)
    ESL_CONF="$FS_CONF_DIR/autoload_configs/event_socket.conf.xml"
    if [[ -f "$ESL_CONF" ]]; then
      # Set ESL password and listen on all interfaces
      sed -i "s|<param name=\"password\" value=\"[^\"]*\"/>|<param name=\"password\" value=\"${FS_PASSWORD}\"/>|" "$ESL_CONF"
      sed -i 's|<param name="listen-ip" value="[^"]*"/>|<param name="listen-ip" value="0.0.0.0"/>|' "$ESL_CONF"
      echo "  ESL password configured."
    fi

    # Configure external SIP profile with EIP
    EXT_SIP="$FS_CONF_DIR/sip_profiles/external.xml"
    if [[ -f "$EXT_SIP" ]]; then
      # Set ext-rtp-ip and ext-sip-ip to EIP
      sed -i "s|<param name=\"ext-rtp-ip\" value=\"[^\"]*\"/>|<param name=\"ext-rtp-ip\" value=\"${EIP}\"/>|" "$EXT_SIP"
      sed -i "s|<param name=\"ext-sip-ip\" value=\"[^\"]*\"/>|<param name=\"ext-sip-ip\" value=\"${EIP}\"/>|" "$EXT_SIP"
      echo "  SIP external profile: ext-rtp-ip/ext-sip-ip = $EIP"
    fi

    # Same for internal profile
    INT_SIP="$FS_CONF_DIR/sip_profiles/internal.xml"
    if [[ -f "$INT_SIP" ]]; then
      sed -i "s|<param name=\"ext-rtp-ip\" value=\"[^\"]*\"/>|<param name=\"ext-rtp-ip\" value=\"${EIP}\"/>|" "$INT_SIP"
      sed -i "s|<param name=\"ext-sip-ip\" value=\"[^\"]*\"/>|<param name=\"ext-sip-ip\" value=\"${EIP}\"/>|" "$INT_SIP"
    fi
  else
    echo "  WARNING: FreeSWITCH config directory not found. Configure manually."
  fi

  # Enable and start FreeSWITCH
  systemctl enable freeswitch 2>/dev/null || true
  systemctl restart freeswitch 2>/dev/null || true

  # Verify ESL connectivity
  sleep 2
  FS_ESL_PORT=$(echo "localhost:8021" | cut -d: -f2)
  if command -v fs_cli &>/dev/null; then
    if fs_cli -H 127.0.0.1 -P "$FS_ESL_PORT" -p "$FS_PASSWORD" -x "status" &>/dev/null; then
      echo "  FreeSWITCH ESL verified."
    else
      echo "  WARNING: fs_cli cannot connect. Check ESL config."
    fi
  fi
fi

# ============================================================
#  [6/10] Deploy BOS3000 binary
# ============================================================
echo "[6/10] Deploying binary..."

id -u bos3000 &>/dev/null || useradd -r -m -d "$INSTALL_DIR" -s /bin/bash bos3000
mkdir -p "$INSTALL_DIR/bin"

cp "$SCRIPT_DIR/bos3000" "$INSTALL_DIR/bin/bos3000"
chmod +x "$INSTALL_DIR/bin/bos3000"
[[ -f "$SCRIPT_DIR/encore-meta" ]] && cp "$SCRIPT_DIR/encore-meta" "$INSTALL_DIR/bin/encore-meta"
cp "$SCRIPT_DIR/reset-admin-password.sh" "$INSTALL_DIR/" 2>/dev/null || true
cp "$SCRIPT_DIR/.version" "$INSTALL_DIR/" 2>/dev/null || true

echo "  Binary: $INSTALL_DIR/bin/bos3000"

# ============================================================
#  [7/10] Write configuration
# ============================================================
echo "[7/10] Writing config..."

JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)

# Determine FS address
if $SKIP_FS; then
  FS_ADDR="<external-fs-host>:8021"
else
  FS_ADDR="localhost:8021"
fi

cat > "$INSTALL_DIR/.env" << ENVEOF
# BOS3000 Production Configuration — Generated $(date -Iseconds)
# Version: $VERSION

# Database
BOS3000_DB_HOST=127.0.0.1
BOS3000_DB_PORT=5432
BOS3000_DB_USER=bos3000
BOS3000_DB_PASSWORD=${DB_PASSWORD}
BOS3000_DB_NAME=auth

# Redis
REDIS_HOST=127.0.0.1
REDIS_PORT=6379

# FreeSWITCH ESL
FS_MODE=real
FS_ADDRESS=${FS_ADDR}
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

# ============================================================
#  [8/10] Firewall
# ============================================================
echo "[8/10] Firewall..."

PORTS_TCP="4000 5060 5061 8021"
PORTS_UDP="5060"
RTP_RANGE="16384:32768"

if command -v ufw &>/dev/null; then
  for p in $PORTS_TCP; do ufw allow "$p/tcp" >/dev/null 2>&1 || true; done
  for p in $PORTS_UDP; do ufw allow "$p/udp" >/dev/null 2>&1 || true; done
  ufw allow "$RTP_RANGE/udp" >/dev/null 2>&1 || true
  echo "  UFW rules added."
elif command -v firewall-cmd &>/dev/null; then
  for p in $PORTS_TCP; do firewall-cmd --permanent --add-port="$p/tcp" >/dev/null 2>&1 || true; done
  for p in $PORTS_UDP; do firewall-cmd --permanent --add-port="$p/udp" >/dev/null 2>&1 || true; done
  firewall-cmd --permanent --add-port="${RTP_RANGE//:/\-}/udp" >/dev/null 2>&1 || true
  firewall-cmd --reload >/dev/null 2>&1 || true
  echo "  firewalld rules added."
else
  echo "  No firewall detected."
fi

# ============================================================
#  [9/10] systemd service
# ============================================================
echo "[9/10] systemd service..."

cat > /etc/systemd/system/bos3000.service << SVCEOF
[Unit]
Description=BOS3000 VoIP Callback Platform
After=network.target postgresql.service redis.service freeswitch.service
Wants=postgresql.service redis.service

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

# ============================================================
#  [10/10] Verification
# ============================================================
echo "[10/10] Pre-flight checks..."

CHECKS_PASSED=0
CHECKS_TOTAL=0

check() {
  local name="$1" cmd="$2"
  CHECKS_TOTAL=$((CHECKS_TOTAL + 1))
  if eval "$cmd" &>/dev/null; then
    echo "  [PASS] $name"
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
  else
    echo "  [FAIL] $name"
  fi
}

check "Binary exists"      "[[ -x $INSTALL_DIR/bin/bos3000 ]]"
check "PostgreSQL running"  "systemctl is-active --quiet postgresql 2>/dev/null || systemctl is-active --quiet postgresql-16 2>/dev/null"
check "DB auth connects"    "PGPASSWORD='$DB_PASSWORD' psql -h 127.0.0.1 -U bos3000 -d auth -c 'SELECT 1'"
check "Redis running"       "redis-cli ping 2>/dev/null | grep -q PONG"
check "systemd unit loaded" "systemctl is-enabled --quiet bos3000"

if ! $SKIP_FS; then
  check "FreeSWITCH running" "systemctl is-active --quiet freeswitch"
fi

check ".env exists"          "[[ -f $INSTALL_DIR/.env ]]"
check "EIP in config"        "grep -q '$EIP' $INSTALL_DIR/.env"

echo ""
echo "  Checks: $CHECKS_PASSED/$CHECKS_TOTAL passed"

# ============================================================
#  Summary
# ============================================================
echo ""
echo "============================================"
echo "  BOS3000 v${VERSION} Installation Complete"
echo "============================================"
echo ""
echo "  Components:"
echo "    PostgreSQL:   $(psql --version 2>/dev/null | head -1 || echo 'installed')"
echo "    Redis:        $(redis-server --version 2>/dev/null | grep -oP 'v=\K[\d.]+' || echo 'installed')"
if ! $SKIP_FS; then
  echo "    FreeSWITCH:   $(freeswitch -version 2>/dev/null || echo 'installed')"
fi
echo "    BOS3000:      v${VERSION}"
echo ""
echo "  Database:"
echo "    Host:         127.0.0.1:5432"
echo "    Databases:    auth, billing, callback, compliance, routing, webhook"
echo "    User:         bos3000"
echo "    Password:     $DB_PASSWORD"
echo ""
echo "  Redis:          127.0.0.1:6379"
echo ""
if ! $SKIP_FS; then
  echo "  FreeSWITCH:"
  echo "    ESL:          localhost:8021 (password: $FS_PASSWORD)"
  echo "    SIP External: $EIP:5060"
  echo ""
fi
echo "  Default Admin:"
echo "    URL:          http://${EIP}:4000/admin/"
echo "    Email:        admin@localhost"
echo "    Password:     changeme123"
echo ""
echo "  Commands:"
echo "    sudo systemctl start bos3000"
echo "    sudo systemctl status bos3000"
echo "    journalctl -u bos3000 -f"
echo "    bash $INSTALL_DIR/reset-admin-password.sh"
echo ""
echo "  IMPORTANT:"
echo "    1. Change the default admin password immediately!"
echo "    2. Save the database password shown above"
echo "    3. Start: sudo systemctl start bos3000"
echo ""
