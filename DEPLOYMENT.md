# SkillHub Deployment Guide

## Prerequisites

Before deploying, set these environment variables on your local machine:

```bash
export LLM_API_KEY="your-openrouter-api-key"
export ADMIN_TOKEN="$(openssl rand -hex 32)"
```

## Deployment Steps

1. **Build and deploy:**
   ```bash
   ./deploy.sh
   ```

2. **Set environment variables on server:**
   ```bash
   ssh root@192.227.235.131
   cd /opt/skillhub
   
   # Set LLM API key
   echo "LLM_API_KEY=your-openrouter-api-key" >> .env
   
   # Set admin token (generate with: openssl rand -hex 32)
   echo "ADMIN_TOKEN=your-strong-random-token" >> .env
   
   # Restart service
   systemctl restart skillhub
   ```

3. **Verify deployment:**
   ```bash
   curl https://skillhub.koolkassanmsk.top/health
   ```

## Environment Variables

### Required
- `DATABASE_URL`: PostgreSQL connection string
- `LLM_API_KEY`: OpenRouter API key for AI review
- `ADMIN_TOKEN`: Strong random token for admin endpoints

### Optional
- `PORT`: Server port (default: 8080)
- `AUTO_MIGRATE`: Run migrations on startup (default: true)
- `SEED_DATA`: Load seed data (default: false)
- `AI_REVIEW_ENABLED`: Enable AI review (default: true)
- `LLM_PROVIDER`: LLM provider (default: openrouter)
- `LLM_MODEL`: LLM model (default: google/gemma-4-31b-it)
- `LLM_BASE_URL`: LLM API base URL
- `SMTP_*`: Email configuration (optional)

## Security Notes

- Never commit API keys or tokens to version control
- Rotate ADMIN_TOKEN regularly
- Use strong random values (32+ bytes)
- Keep .env file permissions restricted (chmod 600)

## Monitoring

- Health check: `https://skillhub.koolkassanmsk.top/health`
- Logs: `ssh root@192.227.235.131 'journalctl -u skillhub -f'`
- Service status: `ssh root@192.227.235.131 'systemctl status skillhub'`

## Rollback

If deployment fails:

```bash
ssh root@192.227.235.131
cd /opt/skillhub
systemctl stop skillhub
# Restore previous binary
mv skillhub.backup skillhub
systemctl start skillhub
```
