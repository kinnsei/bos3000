# BOS3000 Deployment Guide

Single binary deployment — installer handles all dependencies.

## What gets installed

| Component | Version | Purpose |
|-----------|---------|---------|
| PostgreSQL | 16 | 6 databases (auth, billing, callback, compliance, routing, webhook) |
| Redis | 7+ | Concurrent call slots, rate limiting cache |
| FreeSWITCH | 1.10+ | SIP signaling, ESL control (optional: `--skip-freeswitch`) |
| BOS3000 | bundle | Single binary with embedded frontend |

## Quick Start

```bash
# 1. Build on dev machine
bash scripts/build.sh v1.0.0

# 2. Upload to server
scp dist/bos3000-v1.0.0-linux-amd64.tar.gz root@<server>:/tmp/

# 3. Install (installs PG + Redis + FS + binary + systemd)
ssh root@<server>
tar xzf /tmp/bos3000-v1.0.0-linux-amd64.tar.gz -C /tmp
sudo bash /tmp/bos3000/install.sh --eip <your-public-ip>

# 4. Start
sudo systemctl start bos3000
```

## Installer options

```
--eip <ip>           Required. Public IP for SIP signaling
--fs-password <pw>   FreeSWITCH ESL password (default: ClueCon)
--install-dir <dir>  Install path (default: /opt/bos3000)
--skip-freeswitch    Skip FS install (if using external FreeSWITCH)
```

## What the installer does

1. Detects OS (Ubuntu/Debian/CentOS/AlmaLinux)
2. Installs PostgreSQL 16, configures `pg_hba.conf` for password auth
3. Creates 6 databases with random password
4. Installs Redis, verifies PONG
5. Installs FreeSWITCH, configures ESL password + SIP external IP
6. Deploys binary to `/opt/bos3000/bin/bos3000`
7. Generates JWT secret, writes `/opt/bos3000/.env`
8. Configures firewall (ufw/firewalld)
9. Installs systemd service
10. Runs verification checks on all components

## Configuration

After installation, edit `/opt/bos3000/.env`:

| Variable | Description |
|----------|-------------|
| `BOS3000_DB_*` | PostgreSQL connection (auto-configured) |
| `REDIS_HOST/PORT` | Redis connection (default: 127.0.0.1:6379) |
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
journalctl -u bos3000 -f
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
# Build new version
bash scripts/build.sh v1.1.0
scp dist/bos3000-v1.1.0-linux-amd64.tar.gz root@<server>:/tmp/

# On server — just replace binary
tar xzf /tmp/bos3000-v1.1.0-linux-amd64.tar.gz -C /tmp
sudo systemctl stop bos3000
sudo cp /tmp/bos3000/bos3000 /opt/bos3000/bin/bos3000
sudo systemctl start bos3000
```

## External FreeSWITCH

If FreeSWITCH runs on a separate server:

```bash
sudo bash install.sh --eip 47.x.x.x --skip-freeswitch
# Then edit /opt/bos3000/.env:
#   FS_ADDRESS=<fs-server-ip>:8021
#   FS_PASSWORD=<fs-password>
```
