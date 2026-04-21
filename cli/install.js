#!/usr/bin/env node

const { execSync } = require('child_process');
const https = require('https');
const os = require('os');
const path = require('path');
const fs = require('fs');

const AITHUB_API = process.env.AITHUB_API || 'https://aithub.space';
const VERSION = '4.1.0';

// Parse command line arguments
const args = process.argv.slice(2);
const flags = {
  register: args.includes('--register'),
  github: args.includes('--github'),
  google: args.includes('--google'),
  email: args.includes('--email') ? args[args.indexOf('--email') + 1] : null,
  namespace: args.includes('--namespace') ? args[args.indexOf('--namespace') + 1] : null,
  help: args.includes('--help') || args.includes('-h'),
  skipInject: args.includes('--skip-inject')
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

Options:
  --register      Enable registration flow
  --github        Use GitHub OAuth
  --skip-inject   Skip auto-injecting Discovery Skill into AI platforms
  --help, -h      Show this help message

Environment:
  AITHUB_API      Override API URL (default: ${AITHUB_API})
`);
  process.exit(0);
}

console.log('╔══════════════════════════════════════╗');
console.log(`║      AitHub Installer v${VERSION}          ║`);
console.log('║   AI-First Skill Registry + CLI      ║');
console.log('╚══════════════════════════════════════╝');
console.log('');

const homeDir = os.homedir();
const plat = os.platform();
const arch = os.arch();

console.log(`→ OS: ${plat} (${arch})`);
console.log('');

// Determine binary name
let binaryName;
if (plat === 'darwin') {
  binaryName = arch === 'arm64' ? 'aithub-darwin-arm64' : 'aithub-darwin-amd64';
} else if (plat === 'linux') {
  binaryName = arch === 'arm64' ? 'aithub-linux-arm64' : 'aithub-linux-amd64';
} else if (plat === 'win32') {
  binaryName = 'aithub-windows-amd64.exe';
} else {
  console.error(`✗ Unsupported platform: ${plat}`);
  process.exit(1);
}

const downloadUrl = `${AITHUB_API}/downloads/${binaryName}`;
const installDir = path.join(homeDir, '.aithub', 'bin');
const installPath = path.join(installDir, plat === 'win32' ? 'aithub.exe' : 'aithub');

console.log('→ Downloading CLI binary...');
console.log(`  From: ${downloadUrl}`);

fs.mkdirSync(installDir, { recursive: true });

// --- Platform detection ---
function detectPlatforms() {
  const platforms = {};
  const checks = {
    'claude-code':  '.claude',
    'hermes':       '.hermes',
    'openclaw':     '.openclaw',
    'antigravity':  '.gemini',
    'gstack':       '.gstack',
    'cursor':       '.cursor',
    'windsurf':     '.windsurf',
  };
  const skillSubdirs = {
    'claude-code':  '.claude/skills',
    'hermes':       '.hermes/skills',
    'openclaw':     '.openclaw/skills',
    'antigravity':  '.gemini/antigravity/knowledge',
    'gstack':       '.gstack/skills',
    'cursor':       '.cursor/skills',
    'windsurf':     '.windsurf/skills',
  };

  for (const [name, configDir] of Object.entries(checks)) {
    if (fs.existsSync(path.join(homeDir, configDir))) {
      platforms[name] = path.join(homeDir, skillSubdirs[name]);
    }
  }
  return platforms;
}

// --- Inject Discovery Skill into all detected platforms ---
function injectDiscoverySkill(platforms) {
  let discoveryContent;
  try {
    const raw = execSync(
      `curl -s --max-time 10 "${AITHUB_API}/v1/bootstrap/discovery"`,
      { stdio: 'pipe', timeout: 15000 }
    ).toString();
    const data = JSON.parse(raw);
    discoveryContent = data.content;
  } catch (e) {
    console.log('  ⚠ Could not fetch Discovery Skill. Skipping injection.');
    return 0;
  }

  if (!discoveryContent) {
    console.log('  ⚠ Empty Discovery Skill content. Skipping.');
    return 0;
  }

  let count = 0;
  for (const [name, skillDir] of Object.entries(platforms)) {
    try {
      const targetDir = path.join(skillDir, 'aithub-discovery');
      fs.mkdirSync(targetDir, { recursive: true });
      fs.writeFileSync(path.join(targetDir, 'SKILL.md'), discoveryContent);
      console.log(`  ✓ ${name}: injected`);
      count++;
    } catch (e) {
      console.log(`  ⚠ ${name}: failed (${e.message})`);
    }
  }
  return count;
}

// --- Persist credentials to shell profile ---
function persistCredentials(token, namespace) {
  const shellFile = plat === 'win32' ? null :
    fs.existsSync(path.join(homeDir, '.zshrc')) ? '.zshrc' : '.bashrc';
  if (!shellFile) return;

  const shellPath = path.join(homeDir, shellFile);
  try {
    let content = fs.readFileSync(shellPath, 'utf8');
    // Remove old AitHub credentials block
    content = content.replace(/\n# AitHub Credentials\n(export SKILLHUB_\w+=.*\n)*/g, '');
    fs.writeFileSync(shellPath, content);

    let envBlock = '\n# AitHub Credentials\n';
    if (token) envBlock += `export SKILLHUB_TOKEN="${token}"\n`;
    if (namespace) envBlock += `export SKILLHUB_NAMESPACE="${namespace}"\n`;

    if (token || namespace) {
      fs.appendFileSync(shellPath, envBlock);
      console.log(`✓ Credentials persisted to ~/${shellFile}`);
    }
  } catch (e) { /* skip silently */ }
}

// --- Main download flow ---
https.get(downloadUrl, (res) => {
  if (res.statusCode !== 200) {
    console.error(`✗ Download failed: HTTP ${res.statusCode}`);
    process.exit(1);
  }

  const file = fs.createWriteStream(installPath);
  res.pipe(file);

  file.on('finish', () => {
    file.close();
    if (plat !== 'win32') fs.chmodSync(installPath, 0o755);

    console.log(`✓ CLI installed to: ${installPath}`);
    console.log('');

    // Save config
    const configDir = path.join(homeDir, '.aithub');
    fs.mkdirSync(configDir, { recursive: true });
    const configFile = path.join(configDir, 'config.json');
    let config = {};
    try { config = JSON.parse(fs.readFileSync(configFile, 'utf8')); } catch (e) {}
    config.api = AITHUB_API;
    fs.writeFileSync(configFile, JSON.stringify(config, null, 2));
    console.log(`✓ Config: ${configFile}`);

    // Add to PATH
    const shellConfig = plat === 'win32' ? null :
      fs.existsSync(path.join(homeDir, '.zshrc')) ? '.zshrc' : '.bashrc';
    if (shellConfig) {
      const cfgPath = path.join(homeDir, shellConfig);
      const pathLine = `export PATH="$HOME/.aithub/bin:$PATH"`;
      try {
        const c = fs.readFileSync(cfgPath, 'utf8');
        if (!c.includes('.aithub/bin')) {
          fs.appendFileSync(cfgPath, `\n# AitHub CLI\n${pathLine}\n`);
          console.log(`✓ PATH updated in ~/${shellConfig}`);
        }
      } catch (e) {
        console.log(`→ Add to PATH: ${pathLine}`);
      }
    }

    // Test connectivity
    console.log('');
    console.log('→ Testing API...');
    try {
      execSync(`${installPath} search test --limit 1 --api ${AITHUB_API}`, { stdio: 'pipe', timeout: 15000 });
      console.log('✓ API OK');
    } catch (e) {
      console.log('⚠ API test failed. CLI installed but may not reach server.');
    }
    console.log('');

    // --- Auto-detect & inject Discovery Skill ---
    if (!flags.skipInject) {
      console.log('→ Detecting AI platforms...');
      const platforms = detectPlatforms();
      const names = Object.keys(platforms);

      if (names.length > 0) {
        console.log(`  Found: ${names.join(', ')}`);
        console.log('→ Injecting Discovery Skill...');
        const n = injectDiscoverySkill(platforms);
        if (n > 0) console.log(`✓ Discovery Skill injected into ${n} platform(s)`);
      } else {
        console.log('  No AI platforms found. Install Claude Code, Cursor, etc. first.');
      }
      console.log('');
    }

    // --- Registration ---
    if (flags.register && flags.github) {
      console.log('→ Starting GitHub registration...');
      try {
        execSync(`${installPath} register --github --api ${AITHUB_API}`, { stdio: 'inherit', timeout: 900000 });

        // Persist credentials from config.json
        try {
          const cfg = JSON.parse(fs.readFileSync(configFile, 'utf8'));
          if (cfg.token) persistCredentials(cfg.token, cfg.namespace);
        } catch (e) {}
      } catch (e) {
        console.error('✗ Registration failed. Retry later: aithub register --github');
        console.error('  search/install/details work without registration.');
      }
    } else if (flags.register) {
      console.log('⚠ Specify method: npx @aithub/cli --register --github');
    } else {
      // Restore credentials from env/config if available
      try {
        const cfg = JSON.parse(fs.readFileSync(configFile, 'utf8'));
        if (cfg.token) console.log('✓ Existing credentials found');
      } catch (e) {}

      console.log('✓ Installation complete!');
      console.log('');
      console.log('No account needed:');
      console.log('  aithub search <query>              Search skills');
      console.log('  aithub install <namespace/name>    Install a skill');
      console.log('  aithub details <namespace/name>    View skill details');
      console.log('');
      console.log('Unlock rate/submit/fork:');
      console.log('  aithub register --github');
      if (shellConfig) console.log(`\n→ Run: source ~/${shellConfig}`);
    }
  });
}).on('error', (err) => {
  console.error('✗ Download failed:', err.message);
  process.exit(1);
});
