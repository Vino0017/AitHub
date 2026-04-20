#!/usr/bin/env node

const { execSync } = require('child_process');
const https = require('https');
const os = require('os');
const path = require('path');
const fs = require('fs');

const AITHUB_API = process.env.AITHUB_API || 'https://aithub.space';
const VERSION = '3.2.0';

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
║      AitHub CLI Installer v${VERSION}      ║
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
  AITHUB_API      Override API URL (default: ${AITHUB_API})

Examples:
  npx @aithub/cli
  npx @aithub/cli --register --github
  AITHUB_API=http://localhost:8080 npx @aithub/cli
`);
  process.exit(0);
}

console.log('╔══════════════════════════════════════╗');
console.log(`║      AitHub Installer v${VERSION} (npx)      ║`);
console.log('║   AI-First Skill Registry + CLI      ║');
console.log('╚══════════════════════════════════════╝');
console.log('');

// Detect OS and architecture
const platform = os.platform(); // darwin, linux, win32
const arch = os.arch(); // x64, arm64
const homeDir = os.homedir();

console.log(`→ OS: ${platform} (${arch})`);
console.log('');

// Determine binary name
let binaryName;
if (platform === 'darwin') {
  binaryName = arch === 'arm64' ? 'aithub-darwin-arm64' : 'aithub-darwin-amd64';
} else if (platform === 'linux') {
  binaryName = arch === 'arm64' ? 'aithub-linux-arm64' : 'aithub-linux-amd64';
} else if (platform === 'win32') {
  binaryName = 'aithub-windows-amd64.exe';
} else {
  console.error(`✗ Unsupported platform: ${platform}`);
  process.exit(1);
}

// Download binary
const downloadUrl = `${AITHUB_API}/downloads/${binaryName}`;
const installDir = path.join(homeDir, '.aithub', 'bin');
const installPath = path.join(installDir, platform === 'win32' ? 'aithub.exe' : 'aithub');

console.log(`→ Downloading CLI binary...`);
console.log(`  From: ${downloadUrl}`);

// Create install directory
if (!fs.existsSync(installDir)) {
  fs.mkdirSync(installDir, { recursive: true });
}

// Download binary
https.get(downloadUrl, (res) => {
  if (res.statusCode === 404) {
    console.error('✗ Binary not available for your platform');
    console.error('  Please build from source: https://github.com/Vino0017/AitHub');
    process.exit(1);
  }

  if (res.statusCode !== 200) {
    console.error(`✗ Download failed: HTTP ${res.statusCode}`);
    process.exit(1);
  }

  const file = fs.createWriteStream(installPath);
  res.pipe(file);

  file.on('finish', () => {
    file.close();

    // Make executable (Unix only)
    if (platform !== 'win32') {
      fs.chmodSync(installPath, 0o755);
    }

    console.log(`✓ CLI installed to: ${installPath}`);
    console.log('');

    // Add to PATH
    const shellConfig = platform === 'win32' ? null :
                       fs.existsSync(path.join(homeDir, '.zshrc')) ? '.zshrc' : '.bashrc';

    if (shellConfig) {
      const configPath = path.join(homeDir, shellConfig);
      const pathExport = `export PATH="$HOME/.aithub/bin:$PATH"`;

      try {
        const content = fs.readFileSync(configPath, 'utf8');
        if (!content.includes('.aithub/bin')) {
          fs.appendFileSync(configPath, `\n# AitHub CLI\n${pathExport}\n`);
          console.log(`✓ Added to PATH in ~/${shellConfig}`);
        }
      } catch (err) {
        console.log(`→ Add to PATH manually: ${pathExport}`);
      }
    }

    console.log('');

    // Handle registration
    if (flags.register) {
      console.log('→ Starting registration...');
      console.log('');

      let regCmd = `${installPath} config set api ${AITHUB_API}`;

      try {
        execSync(regCmd, { stdio: 'inherit' });
      } catch (err) {
        console.error('✗ Registration failed');
        process.exit(1);
      }

      console.log('');
      console.log('✓ Registration complete!');
      console.log('');
      console.log('Next steps:');
      console.log('  1. Restart your terminal (or run: source ~/' + shellConfig + ')');
      console.log('  2. Search skills: aithub search <query>');
      console.log('  3. Install a skill: aithub install <namespace/name> --deploy');
      console.log('');
      console.log('Documentation: https://aithub.space/docs');
    } else {
      console.log('✓ Installation complete!');
      console.log('');
      console.log('Next steps:');
      console.log('  1. Restart your terminal (or run: source ~/' + (shellConfig || '.bashrc') + ')');
      console.log('  2. Register (optional): npx @aithub/cli --register --github');
      console.log('  3. Search skills: aithub search <query>');
      console.log('');
      console.log('Documentation: https://aithub.space/docs');
    }
  });
}).on('error', (err) => {
  console.error('✗ Download failed:', err.message);
  console.error('');
  console.error('Fallback: Install manually from GitHub releases');
  console.error('  https://github.com/Vino0017/AitHub/releases');
  process.exit(1);
});
