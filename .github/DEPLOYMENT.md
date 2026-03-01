# NodeByte Backend Deployment Guide

Complete guide for deploying the NodeByte Backend Go API service on Ubuntu using systemd and Nginx with Cloudflare Origin Certificates.

**Service:** Backend API (Go 1.24+)  
**Environment:** Ubuntu 20.04 LTS or 22.04 LTS  
**Date:** February 2026

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [System Setup](#system-setup)
3. [Backend Deployment](#backend-deployment)
4. [Nginx Configuration](#nginx-configuration)
5. [SSL/TLS Setup](#ssltls-setup)
6. [Monitoring](#monitoring)
7. [Maintenance](#maintenance)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software
- Ubuntu 20.04 LTS or 22.04 LTS
- Go 1.24+
- PostgreSQL 12+
- Redis 6+
- Nginx 1.18+
- Git

### Required Accounts
- Cloudflare account with domain configured
- Access to server with sudo privileges

### Domain Assumption
- Backend API: `api.yourdomain.com`

---

## System Setup

### 1. Update System
```bash
sudo apt update
sudo apt upgrade -y
```

### 2. Install Dependencies

#### Go 1.24+
```bash
# Download latest Go 1.24
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
rm go1.24.linux-amd64.tar.gz

# Add to system-wide PATH
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh

# Verify installation
go version
```

#### PostgreSQL & Redis
```bash
sudo apt install -y postgresql postgresql-contrib redis-server
```

#### Nginx
```bash
sudo apt install -y nginx
```

#### Additional Tools
```bash
sudo apt install -y git curl wget htop ufw
```

### 3. Create Deploy User
```bash
sudo useradd -m -s /bin/bash deploy
sudo usermod -aG sudo deploy
```

### 4. Setup Firewall
```bash
sudo ufw enable
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

### 5. Create Application Directory
```bash
sudo mkdir -p /var/www/nodebyte/backend
sudo chown -R deploy:deploy /var/www/nodebyte
```

---

## Backend Deployment

### 1. Clone Repository
```bash
cd /var/www/nodebyte
sudo -u deploy git clone <backend-repo-url> backend
cd backend
sudo -u deploy git checkout v0.2.2-testing  # or your desired branch
```

### 2. Setup Environment Variables
```bash
sudo -u deploy cp .env.example .env
sudo -u deploy nano .env
```

Configure `.env`:
```env
# Server
PORT=8080
ENV=production

# Database
DATABASE_URL=postgres://deploy:secure-password@localhost:5432/nodebyte_prod
REDIS_URL=redis://localhost:6379

# Logging
LOG_LEVEL=info

# Authentication
JWT_SECRET=your-secure-jwt-secret-here

# External APIs (configure as needed)
HYTALE_CLIENT_ID=your-client-id
HYTALE_CLIENT_SECRET=your-client-secret
SENTRY_DSN=your-sentry-dsn

# Email (Resend API)
RESEND_API_KEY=your-resend-api-key
RESEND_FROM_EMAIL=noreply@yourdomain.com

# Webhooks
WEBHOOK_SECRET=your-webhook-secret

# Pterodactyl Integration (optional)
PTERODACTYL_URL=https://panel.yourdomain.com
PTERODACTYL_API_KEY=your-admin-api-key
PTERODACTYL_CLIENT_API_KEY=your-client-api-key

# Encryption (optional - for sensitive data)
ENCRYPTION_KEY=your-32-byte-hex-key
```

### 3. Setup Database
```bash
# As postgres user
sudo -u postgres createdb nodebyte_prod
sudo -u postgres createuser deploy
sudo -u postgres psql -c "ALTER USER deploy WITH PASSWORD 'secure-password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE nodebyte_prod TO deploy;"

# Grant permissions on public schema
sudo -u postgres psql -d nodebyte_prod -c "GRANT ALL PRIVILEGES ON SCHEMA public TO deploy;"
sudo -u postgres psql -d nodebyte_prod -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO deploy;"
sudo -u postgres psql -d nodebyte_prod -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO deploy;"

# Verify connection
sudo -u postgres psql -c "SELECT 1" -d nodebyte_prod
```

### 4. Initialize Database

**Note:** Schema initialization has dependency ordering issues. Run schemas manually:

```bash
cd /var/www/nodebyte/backend

# Build database tools
sudo -u deploy make build-tools

# Run schemas in correct dependency order
sudo -u deploy make db-migrate-schema SCHEMA=schema_01_users_auth.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_03_servers.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_02_pterodactyl_sync.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_04_billing.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_05_support_tickets.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_06_discord_webhooks.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_07_sync_logs.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_08_config.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_09_hytale.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_10_hytale_audit.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_11_hytale_server_logs.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_12_server_subusers.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_13_hytale_server_link.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_14_partners.sql
sudo -u deploy make db-migrate-schema SCHEMA=schema_15_careers.sql

# Verify all schemas loaded
sudo -u deploy make db-list
```

### 5. Build Backend
```bash
cd /var/www/nodebyte/backend
sudo -u deploy make build

# Verify binary
ls -lh ./bin/nodebyte-backend
```

### 6. Create Systemd Service

Create `/etc/systemd/system/nodebyte-backend.service`:

```ini
[Unit]
Description=NodeByte Backend API Service
After=network.target postgresql.service redis-server.service
Wants=postgresql.service redis-server.service

[Service]
Type=simple
User=deploy
WorkingDirectory=/var/www/nodebyte/backend
Environment="PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin"
ExecStart=/var/www/nodebyte/backend/bin/nodebyte-backend
Restart=on-failure
RestartSec=30
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nodebyte-backend

# Performance tuning
LimitNOFILE=65000
LimitNPROC=65000

# Security
PrivateTmp=yes
NoNewPrivileges=yes

[Install]
WantedBy=multi-user.target
```

### 7. Enable and Start Service
```bash
sudo systemctl daemon-reload
sudo systemctl enable nodebyte-backend
sudo systemctl start nodebyte-backend

# Verify
sudo systemctl status nodebyte-backend
sudo journalctl -u nodebyte-backend -n 50
```

---

## Nginx Configuration

### 1. Create Nginx Configuration

Create `/etc/nginx/sites-available/nodebyte-backend`:

```nginx
upstream backend {
    server 127.0.0.1:8080 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

server {
    listen 80;
    listen [::]:80;
    server_name api.yourdomain.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name api.yourdomain.com;

    # SSL certificates (Cloudflare Origin Certificates)
    ssl_certificate /etc/ssl/certs/yourdomain.com.crt;
    ssl_certificate_key /etc/ssl/private/yourdomain.com.key;

    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Logging
    access_log /var/log/nginx/api-access.log combined;
    error_log /var/log/nginx/api-error.log warn;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=30r/s;
    limit_req zone=api_limit burst=50 nodelay;

    location / {
        proxy_pass http://backend;
        proxy_http_version 1.1;

        # Headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Buffering
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
        proxy_busy_buffers_size 8k;
    }

    location /health {
        access_log off;
        proxy_pass http://backend;
    }
}
```

### 2. Install Cloudflare Origin Certificates

```bash
# Create certificates directory
sudo mkdir -p /etc/ssl/certs /etc/ssl/private

# Create certificate file (get from Cloudflare Dashboard → SSL/TLS → Origin Server)
sudo tee /etc/ssl/certs/yourdomain.com.crt > /dev/null << 'EOF'
-----BEGIN CERTIFICATE-----
(Paste your Cloudflare certificate here)
-----END CERTIFICATE-----
EOF

# Create private key file
sudo tee /etc/ssl/private/yourdomain.com.key > /dev/null << 'EOF'
-----BEGIN PRIVATE KEY-----
(Paste your private key here)
-----END PRIVATE KEY-----
EOF

# Set correct permissions
sudo chmod 644 /etc/ssl/certs/yourdomain.com.crt
sudo chmod 600 /etc/ssl/private/yourdomain.com.key
sudo chown root:root /etc/ssl/certs/yourdomain.com.crt /etc/ssl/private/yourdomain.com.key
```

### 3. Enable Nginx Site
```bash
sudo ln -s /etc/nginx/sites-available/nodebyte-backend /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx
sudo systemctl enable nginx
```

---

## SSL/TLS Setup

### Cloudflare Configuration

1. **SSL/TLS → Overview**
   - Mode: Full (strict)
   - Minimum TLS Version: TLS 1.2

2. **Edge Certificates**
   - Always use HTTPS: On
   - HSTS: Enable (max-age=31536000)
   - TLS 1.3: On

3. **DNS**
   - Proxy status: Proxied (orange cloud)

---

## Monitoring

### View Service Logs
```bash
# Real-time logs
sudo journalctl -u nodebyte-backend -f

# Last 50 lines
sudo journalctl -u nodebyte-backend -n 50

# Filter by time
sudo journalctl -u nodebyte-backend --since "2 hours ago"
```

### Service Status
```bash
sudo systemctl status nodebyte-backend
```

### Health Check
```bash
curl -k https://api.yourdomain.com/health
```

### System Resources
```bash
# CPU/Memory usage
htop -p $(pgrep nodebyte-backend)

# Disk usage
df -h /var/www/nodebyte

# Network connections
sudo ss -tulpn | grep 8080
```

---

## Maintenance

### Updating Backend

```bash
cd /var/www/nodebyte/backend

# Fetch latest changes
sudo -u deploy git fetch origin
sudo -u deploy git pull origin v0.2.2-testing

# Update dependencies
sudo -u deploy go mod download
sudo -u deploy go mod tidy

# Rebuild
sudo -u deploy make build

# Restart service
sudo systemctl restart nodebyte-backend

# Verify
sudo systemctl status nodebyte-backend
sudo journalctl -u nodebyte-backend -n 20
```

### Database Migrations

```bash
cd /var/www/nodebyte/backend

# Backup database first
sudo -u postgres pg_dump nodebyte_prod > /backup/nodebyte_$(date +%Y%m%d_%H%M%S).sql

# Stop service
sudo systemctl stop nodebyte-backend

# Run migrations
sudo -u deploy make db-migrate

# Restart service
sudo systemctl start nodebyte-backend
```

### Rollback
```bash
cd /var/www/nodebyte/backend

# View recent commits
sudo -u deploy git log --oneline -10

# Checkout previous version
sudo -u deploy git checkout <commit-hash>

# Rebuild and restart
sudo -u deploy make build
sudo systemctl restart nodebyte-backend
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u nodebyte-backend -n 50

# Verify environment variables
cat /var/www/nodebyte/backend/.env

# Test database connection
sudo -u deploy psql postgresql://deploy:password@localhost:5432/nodebyte_prod -c "SELECT 1"

# Check if port is in use
sudo lsof -i :8080

# Verify binary
ls -la /var/www/nodebyte/backend/bin/nodebyte-backend
./bin/nodebyte-backend --version 2>/dev/null || echo "Binary exists"
```

### Database Connection Issues

```bash
# Test connection
sudo -u postgres psql -d nodebyte_prod -c "SELECT version();"

# Check PostgreSQL status
sudo systemctl status postgresql

# View PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-*.log
```

### Redis Connection Issues

```bash
# Test Redis
redis-cli ping

# Check Redis status
sudo systemctl status redis-server

# View Redis logs
sudo journalctl -u redis-server -n 50
```

### Nginx Issues

```bash
# Test configuration
sudo nginx -t

# Check Nginx logs
sudo tail -f /var/log/nginx/api-error.log

# Verify upstream is accessible
curl http://localhost:8080/health
```

### Performance Issues

```bash
# Check system resources
free -h
df -h
top

# Check database connections
sudo -u postgres psql -d nodebyte_prod -c "SELECT count(*) FROM pg_stat_activity;"

# Check Redis memory
redis-cli info memory
```

---

## Quick Reference Commands

```bash
# Service management
sudo systemctl start nodebyte-backend
sudo systemctl stop nodebyte-backend
sudo systemctl restart nodebyte-backend
sudo systemctl status nodebyte-backend

# Logs
sudo journalctl -u nodebyte-backend -f
sudo tail -f /var/log/nginx/api-access.log

# Health checks
curl http://localhost:8080/health
curl -k https://api.yourdomain.com/health

# Database
sudo -u postgres psql -d nodebyte_prod
sudo -u deploy make db-list

# Rebuild
cd /var/www/nodebyte/backend
sudo -u deploy make build
sudo systemctl restart nodebyte-backend
```

---

**Last Updated:** February 28, 2026
