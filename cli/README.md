# AitHub NPX Installer

Install AitHub with a single command using npx:

```bash
npx @aithub/cli
```

## Features

- ✅ Cross-platform (macOS, Linux, Windows)
- ✅ No manual script downloads
- ✅ Automatic bash version detection on macOS
- ✅ Registration flow support
- ✅ Framework auto-detection and deployment

## Usage

### Anonymous Install (Search & Install Only)

```bash
npx @aithub/cli
```

### Register with GitHub OAuth

```bash
npx @aithub/cli --register --github
```

### Register with Email

```bash
npx @aithub/cli --register --email your@email.com --namespace yourname
```

### Custom API URL

```bash
SKILLHUB_API=http://localhost:8080 npx @aithub/cli
```

## Options

- `--register` - Enable registration flow
- `--github` - Use GitHub OAuth (requires `--register`)
- `--google` - Use Google OAuth (requires `--register`)
- `--email <addr>` - Use email verification (requires `--register`)
- `--namespace <name>` - Set namespace for email registration
- `--help`, `-h` - Show help message

## What Gets Installed

1. **aithub CLI** - Command-line tool for searching, installing, and managing skills
2. **Discovery Skill** - `/skillhub` skill for AI agents
3. **Routing Rules** - Auto-search SkillHub before solving complex tasks
4. **Environment Config** - `SKILLHUB_TOKEN` and PATH setup

## Supported AI Frameworks

- Claude Code
- Cursor
- Windsurf
- gstack
- OpenClaw
- Hermes

## Publishing to npm

```bash
cd cli
npm publish --access public
```

## Local Testing

```bash
cd cli
npm link
skillhub-install --help
```

Or test directly:

```bash
node cli/install.js --help
```
