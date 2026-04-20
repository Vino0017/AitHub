#!/usr/bin/env node

const { spawn } = require('child_process');
const https = require('https');
const os = require('os');
const path = require('path');
const fs = require('fs');

const SKILLHUB_API = process.env.SKILLHUB_API || 'https://aithub.space';

// Parse command line arguments
const args = process.argv.slice(2);
const flags = {
  register: args.includes('--register'),
  github: args.includes('--github'),
  google: args.includes('--google'),
  email: args.includes('--email') ? args[args.indexOf('--email') + 1] : null,
  namespace: args.includes('--namespace') ? args[args.indexOf('--namespace') + 1] : null,
  help: args.includes('--help') || args.includes('-h')
};

if (flags.help) {
  console.log(`
╔══════════════════════════════════════╗
║      AitHub CLI Installer v3         ║
║   AI-First Skill Registry + CLI      ║
╚══════════════════════════════════════╝

Usage:
  npx @aithub/cli                           # Anonymous install
  npx @aithub/cli --register --github       # Register with GitHub
  npx @aithub/cli --register --google       # Register with Google
  npx @aithub/cli --register --email <email> --namespace <name>

Options:
  --register      Enable registration flow
  --github        Use GitHub OAuth
  --google        Use Google OAuth
  --email <addr>  Use email verification
  --namespace <n> Set namespace for email registration
  --help, -h      Show this help message

Environment:
  SKILLHUB_API    Override API URL (default: ${SKILLHUB_API})

Examples:
  npx @aithub/cli
  npx @aithub/cli --register --github
  SKILLHUB_API=http://localhost:8080 npx @aithub/cli
`);
  process.exit(0);
}

console.log('╔══════════════════════════════════════╗');
console.log('║      AitHub Installer v3 (npx)       ║');
console.log('║   AI-First Skill Registry + CLI      ║');
console.log('╚══════════════════════════════════════╝');
console.log('');

// Detect OS and architecture
const platform = os.platform(); // darwin, linux, win32
const arch = os.arch(); // x64, arm64
const homeDir = os.homedir();

console.log(`→ OS: ${platform} (${arch})`);
console.log('');

// Download and execute the bash/powershell installer
const installScriptUrl = `${SKILLHUB_API}/install`;

if (platform === 'win32') {
  // Windows: download and run PowerShell script
  console.log('→ Downloading Windows installer...');
  https.get(`${SKILLHUB_API}/install.ps1`, (res) => {
    let script = '';
    res.on('data', (chunk) => script += chunk);
    res.on('end', () => {
      const tempFile = path.join(os.tmpdir(), 'skillhub-install.ps1');
      fs.writeFileSync(tempFile, script);

      const psArgs = ['-ExecutionPolicy', 'Bypass', '-File', tempFile];
      if (flags.register) psArgs.push('-Register');
      if (flags.github) psArgs.push('-GitHub');
      if (flags.google) psArgs.push('-Google');
      if (flags.email) psArgs.push('-Email', flags.email);
      if (flags.namespace) psArgs.push('-Namespace', flags.namespace);

      const ps = spawn('powershell.exe', psArgs, { stdio: 'inherit' });
      ps.on('close', (code) => {
        fs.unlinkSync(tempFile);
        process.exit(code);
      });
    });
  }).on('error', (err) => {
    console.error('✗ Failed to download installer:', err.message);
    process.exit(1);
  });
} else {
  // Unix: download and run bash script
  console.log('→ Downloading Unix installer...');
  https.get(installScriptUrl, (res) => {
    let script = '';
    res.on('data', (chunk) => script += chunk);
    res.on('end', () => {
      const bashArgs = [];
      if (flags.register) bashArgs.push('--register');
      if (flags.github) bashArgs.push('--github');
      if (flags.google) bashArgs.push('--google');
      if (flags.email) {
        bashArgs.push('--email', flags.email);
        if (flags.namespace) bashArgs.push('--namespace', flags.namespace);
      }

      // Try to find bash 4.0+ first
      let bashPath = '/bin/bash';
      if (platform === 'darwin') {
        if (fs.existsSync('/opt/homebrew/bin/bash')) {
          bashPath = '/opt/homebrew/bin/bash';
        } else if (fs.existsSync('/usr/local/bin/bash')) {
          bashPath = '/usr/local/bin/bash';
        }
      }

      const bash = spawn(bashPath, bashArgs, {
        stdio: ['pipe', 'inherit', 'inherit'],
        env: { ...process.env, SKILLHUB_API }
      });

      bash.stdin.write(script);
      bash.stdin.end();

      bash.on('close', (code) => {
        process.exit(code);
      });
    });
  }).on('error', (err) => {
    console.error('✗ Failed to download installer:', err.message);
    console.error('');
    console.error('Fallback: Run directly with curl:');
    console.error(`  curl -fsSL ${installScriptUrl} | bash`);
    process.exit(1);
  });
}
