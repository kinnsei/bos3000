# BOS3000 Deployment Guide

## Quick Start

```bash
# 1. Build
bash scripts/build.sh v1.0.0

# 2. Upload to server
scp dist/bos3000-v1.0.0.tar.gz root@<server>:/tmp/

# 3. Install on server
ssh root@<server>
tar xzf /tmp/bos3000-v1.0.0.tar.gz -C /tmp/bos3000-deploy
cd /tmp/bos3000-deploy
sudo bash deploy/install.sh --version v1.0.0 --eip <your-public-ip>

# 4. Start
sudo systemctl start bos3000
```

## Prerequisites

- Ubuntu 22.04/24.04 or CentOS 8+ / AlmaLinux 8+
- Root access
- Elastic IP (EIP) for SIP signaling
- FreeSWITCH (external, or install separately)

## Configuration

After installation, edit `/opt/bos3000/.env`:

| Variable | Description |
|----------|-------------|
| `BOS3000_DB_*` | PostgreSQL connection (auto-configured by installer) |
| `FS_ADDRESS` | FreeSWITCH ESL host:port |
| `FS_PASSWORD` | FreeSWITCH ESL password |
| `JWT_SECRET` | JWT signing key (auto-generated) |
| `EIP` | Public IP for SIP signaling |

## Default Admin

- Email: `admin@localhost`
- Password: `changeme123`
- **Change immediately after first login!**

## Password Reset

If you forget the admin password:

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
bash scripts/build.sh v1.1.0
scp dist/bos3000-v1.1.0.tar.gz root@<server>:/tmp/
ssh root@<server>
sudo systemctl stop bos3000
# Deploy new binary to /opt/bos3000/bin/
sudo systemctl start bos3000
```
