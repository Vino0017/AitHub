# SkillHub Test Server Deployment Guide

## Quick Deploy (Docker Compose)

### Prerequisites
- Ubuntu 20.04+ / Debian 11+
- Docker & Docker Compose installed
- Domain name (optional, can use IP)
- 2GB+ RAM, 20GB+ disk

### 1. Clone Repository

```bash
git clone https://github.com/Vino0017/AitHub.git
cd AitHub
```

### 2. Configure Environment

```bash
cp .env.example .env
nano .env
```

Required settings:
```env
# Database
DATABASE_URL=postgres://skillhub:your_password@postgres:5432/skillhub?sslmode=disable

# API
PORT=8080
DOMAIN=https://your-test-domain.com  # or http://your-ip:8080

# Auto-migrate on startup
AUTO_MIGRATE=true

# Optional: Email (for registration)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@your-domain.com

# Optional: GitHub OAuth (for GitHub login)
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
```

### 3. Start Services

```bash
docker compose up -d
```

This will start:
- PostgreSQL database (port 5432, localhost only)
- API server (port 8080)
- Next.js web frontend (port 3000, localhost only)

### 4. Verify Installation

```bash
# Check services
docker compose ps

# Check API health
curl http://localhost:8080/health

# Check stats
curl http://localhost:8080/v1/stats

# View logs
docker compose logs -f api
```

### 5. Setup Nginx (Optional, for HTTPS)

```nginx
server {
    listen 80;
    server_name your-test-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable HTTPS with Let's Encrypt:
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-test-domain.com
```

### 6. Test Installation

```bash
# Test from another machine
curl https://your-test-domain.com/health

# Install CLI
bash <(curl -fsSL https://your-test-domain.com/install)

# Test CLI
aithub search "test"
```

## Manual Deployment (Without Docker)

### Prerequisites
- Go 1.23+
- PostgreSQL 17+
- Node.js 20+ (for web frontend)

### 1. Setup Database

```bash
sudo -u postgres psql
CREATE DATABASE skillhub;
CREATE USER skillhub WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE skillhub TO skillhub;
\c skillhub
CREATE EXTENSION IF NOT EXISTS vector;
\q
```

### 2. Build API

```bash
cd AitHub
go build -o skillhub ./cmd/api
```

### 3. Run Migrations

```bash
export DATABASE_URL="postgres://skillhub:your_password@localhost:5432/skillhub?sslmode=disable"
./skillhub migrate
```

### 4. Start API

```bash
export PORT=8080
export DOMAIN=https://your-test-domain.com
./skillhub
```

### 5. Build CLI Binaries

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o dist/aithub-linux-amd64 ./cmd/aithub

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o dist/aithub-linux-arm64 ./cmd/aithub

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o dist/aithub-darwin-amd64 ./cmd/aithub

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o dist/aithub-darwin-arm64 ./cmd/aithub

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o dist/aithub-windows-amd64.exe ./cmd/aithub
```

## Preload Skills

```bash
# Download popular skill collections
bash scripts/preload_popular_skills.sh

# Or manually add skills
aithub submit /path/to/SKILL.md
```

## Troubleshooting

### Database Connection Failed
```bash
# Check PostgreSQL is running
sudo systemctl status postgresql

# Check connection
psql $DATABASE_URL -c "SELECT 1"
```

### API Not Starting
```bash
# Check logs
docker compose logs api

# Check port availability
sudo netstat -tlnp | grep 8080
```

### CLI Download 404
```bash
# Verify binaries exist
ls -la dist/

# Check Docker volume mount
docker compose exec api ls -la /app/dist/
```

### Routing Rules Not Installing
```bash
# Check framework directories
ls -la ~/.claude/rules/common/
ls -la ~/.hermes/

# Manually install
curl https://your-domain.com/install | bash
```

## Production Checklist

- [ ] Change default database password
- [ ] Enable HTTPS with valid certificate
- [ ] Configure firewall (allow 80, 443, block 5432, 3000)
- [ ] Setup backup for PostgreSQL
- [ ] Configure log rotation
- [ ] Setup monitoring (health checks)
- [ ] Configure rate limiting
- [ ] Review security settings
- [ ] Test disaster recovery

## Monitoring

### Health Check
```bash
curl https://your-domain.com/health
# Expected: {"ok":true,"version":"2.0.0"}
```

### Stats
```bash
curl https://your-domain.com/v1/stats
# Expected: {"total_contributors":N,"total_installs":N,"total_skills":N}
```

### Database Size
```bash
docker compose exec postgres psql -U skillhub -d skillhub -c "
SELECT pg_size_pretty(pg_database_size('skillhub'));"
```

### Container Resources
```bash
docker stats skillhub-api-1 skillhub-postgres-1
```

## Backup & Restore

### Backup Database
```bash
docker compose exec postgres pg_dump -U skillhub skillhub > backup-$(date +%Y%m%d).sql
```

### Restore Database
```bash
cat backup-20260419.sql | docker compose exec -T postgres psql -U skillhub skillhub
```

## Updating

### Update Code
```bash
git pull origin main
docker compose build
docker compose up -d
```

### Update CLI Binaries
```bash
# Rebuild all platforms
bash scripts/build_cli.sh

# Or rebuild in Docker
docker compose build api
docker compose restart api
```

## Support

- GitHub Issues: https://github.com/Vino0017/AitHub/issues
- Documentation: https://your-domain.com/docs
- API Reference: https://your-domain.com/docs
