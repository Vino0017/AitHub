#!/usr/bin/env node

const { execSync } = require('child_process');
const https = require('https');
const os = require('os');
const path = require('path');
const fs = require('fs');

const AITHUB_API = process.env.AITHUB_API || 'https://aithub.space';
const VERSION = '4.0.0';

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

    // Save base config
    const configDir = path.join(homeDir, '.aithub');
    if (!fs.existsSync(configDir)) {
      fs.mkdirSync(configDir, { recursive: true });
    }
    const configFile = path.join(configDir, 'config.json');
    let config = {};
    try { config = JSON.parse(fs.readFileSync(configFile, 'utf8')); } catch (e) {}
    config.api = AITHUB_API;
    fs.writeFileSync(configFile, JSON.stringify(config, null, 2));
    console.log(`✓ Config saved to: ${configFile}`);

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

    // Test API connectivity
    console.log('');
    console.log('→ Testing API connectivity...');
    try {
      execSync(`${installPath} search test --limit 1 --api ${AITHUB_API}`, { stdio: 'pipe', timeout: 15000 });
      console.log('✓ API connection OK');
    } catch (err) {
      console.log('⚠ API test failed — CLI installed but may not connect to server.');
      console.log(`  Check: ${AITHUB_API}/health`);
    }

    console.log('');

    // Handle registration
    if (flags.register && flags.github) {
      console.log('→ Starting GitHub registration...');
      console.log('  (This will open a GitHub device authorization flow)');
      console.log('');

      try {
        execSync(`${installPath} register --github --api ${AITHUB_API}`, { stdio: 'inherit', timeout: 900000 });
      } catch (err) {
        console.error('');
        console.error('✗ Registration failed. You can retry later:');
        console.error(`  ${installPath} register --github`);
        console.error('');
        console.error('Note: search/install/details work without registration.');
      }
    } else if (flags.register) {
      console.log('⚠ Please specify a registration method:');
      console.log('  npx @aithub/cli --register --github');
    } else {
      console.log('✓ Installation complete!');
      console.log('');
      console.log('Usage (no account needed):');
      console.log('  aithub search <query>              Search skills');
      console.log('  aithub install <namespace/name>    Install a skill');
      console.log('  aithub details <namespace/name>    View skill details');
      console.log('');
      console.log('To unlock rate/submit/fork:');
      console.log('  aithub register --github');
      console.log('');
      if (shellConfig) {
        console.log(`→ Restart your terminal or run: source ~/${shellConfig}`);
      }
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
