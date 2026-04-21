# GitHub OAuth Setup Guide

## Problem
GitHub OAuth returns "github_not_configured" because `GITHUB_CLIENT_ID` is not set.

## Solution Options

### Option 1: Enable GitHub OAuth (Recommended)

1. **Create GitHub OAuth App**
   - Go to: https://github.com/settings/developers
   - Click "New OAuth App"
   - Fill in:
     - Application name: `SkillHub`
     - Homepage URL: `https://aithub.space`
     - Authorization callback URL: (leave empty - device flow doesn't use it)
   - Click "Register application"
   - Copy the **Client ID** and generate a **Client Secret**

2. **Configure Environment Variables**
   ```bash
   # On your production server
   export GITHUB_CLIENT_ID="your_client_id_here"
   export GITHUB_CLIENT_SECRET="your_client_secret_here"
   ```

3. **Restart Docker Container**
   ```bash
   docker compose down
   docker compose up -d
   ```

4. **Test Registration**
   ```bash
   curl -X POST https://aithub.space/v1/auth/github
   # Should return device_code and user_code
   ```

### Option 2: Use Email Verification Instead

If you don't want to set up GitHub OAuth, use email verification:

```bash
# Install with email registration
npx @skillhub/cli --register --email your@email.com --namespace yourname
```

This requires SMTP configuration or `SKILLHUB_DEV_MODE=true` for development.

### Option 3: Disable Registration Prompts

Remove registration prompts from install script and make it purely optional:

```bash
# Simple install (anonymous token)
npx @skillhub/cli

# Register later when needed
aithub register --email your@email.com --namespace yourname
```

## Current Status

- ✅ GitHub OAuth code is implemented (device flow)
- ❌ GitHub OAuth credentials not configured on server
- ✅ Email verification is implemented
- ❌ SMTP not configured (can use dev mode)
- ✅ Anonymous tokens work for search/install

## Recommendation

For production: Set up GitHub OAuth (Option 1)
For testing: Use `SKILLHUB_DEV_MODE=true` with email verification
For simplicity: Remove registration prompts (Option 3)
