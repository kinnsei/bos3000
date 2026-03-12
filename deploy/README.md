# BOS3000 Deployment Guide

Single binary deployment — no Docker required.

## Quick Start

```bash
# 1. Build on dev machine
bash scripts/build.sh v1.0.0

# 2. Upload to server
scp dist/bos3000-v1.0.0-linux-amd64.tar.gz root@<server>:/tmp/

# 3. Install on server
ssh root@<server>
tar xzf /tmp/bos3000-v1.0.0-linux-amd64.tar.gz -C /tmp
sudo bash /tmp/bos3000/install.sh --eip <your-public-ip>

# 4. Start
sudo systemctl start bos3000
```

## What the installer does

1. Installs PostgreSQL 16 (if not present)
2. Creates database `bos3000` with random password
3. Creates system user `bos3000`
4. Copies binary to `/opt/bos3000/bin/bos3000`
5. Generates JWT secret, writes `/opt/bos3000/.env`
6. Configures firewall (ufw or firewalld)
7. Installs systemd service

## Prerequisites

- Ubuntu 22.04/24.04 or CentOS 8+ / AlmaLinux 8+
- Root access
- Elastic IP (EIP) for SIP signaling
- FreeSWITCH (external, or install separately)

## Bundle contents

```
bos3000/
├── bos3000                    # Linux amd64 binary (single file)
├── encore-meta                # Encore runtime metadata
├── install.sh                 # Installer script
├── reset-admin-password.sh    # Password reset utility
├── env.example                # Environment variable reference
├── bos3000.service            # systemd unit (reference only, installer generates its own)
├── README.md                  # This file
└── .version                   # Version string
```

## Configuration

After installation, edit `/opt/bos3000/.env`:

| Variable | Description |
|----------|-------------|
| `BOS3000_DB_*` | PostgreSQL connection (auto-configured) |
| `FS_ADDRESS` | FreeSWITCH ESL host:port |
| `FS_PASSWORD` | FreeSWITCH ESL password |
| `JWT_SECRET` | JWT signing key (auto-generated) |
| `EIP` | Public IP for SIP signaling |

## Default Admin

- Email: `admin@localhost`
- Password: `changeme123`
- **Change immediately after first login!**

## Password Reset

```bash
bash /opt/bos3000/reset-admin-password.sh
```

## Service Management

```bash
sudo systemctl start bos3000
sudo systemctl stop bos3000
sudo systemctl restart bos3000
sudo systemctl status bos3000
journalctl -u bos3000 -f    # View logs
```

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 4000 | TCP | API + Web UI |
| 5060 | UDP/TCP | SIP Signaling |
| 5061 | TCP | SIP TLS |
| 8021 | TCP | FreeSWITCH ESL |
| 16384-32768 | UDP | RTP Media |

## Upgrading

```bash
# On dev machine
bash scripts/build.sh v1.1.0
scp dist/bos3000-v1.1.0-linux-amd64.tar.gz root@<server>:/tmp/

# On server
ssh root@<server>
tar xzf /tmp/bos3000-v1.1.0-linux-amd64.tar.gz -C /tmp
sudo systemctl stop bos3000
sudo cp /tmp/bos3000/bos3000 /opt/bos3000/bin/bos3000
sudo systemctl start bos3000
```
