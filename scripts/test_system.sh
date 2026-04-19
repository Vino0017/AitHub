#!/usr/bin/env bash
# Test SkillHub CLI + Auto-Discovery System

set -euo pipefail

echo "╔══════════════════════════════════════╗"
echo "║  SkillHub CLI System Test           ║"
echo "╚══════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() {
  echo -e "${GREEN}✓${NC} $1"
}

fail() {
  echo -e "${RED}✗${NC} $1"
  exit 1
}

warn() {
  echo -e "${YELLOW}⚠${NC} $1"
}

# Test 1: CLI binary exists
echo "Test 1: CLI binary exists"
if [ -f "./dist/aithub-darwin-arm64" ]; then
  pass "CLI binary found"
else
  fail "CLI binary not found. Run ./scripts/build_cli.sh first"
fi

# Test 2: CLI version
echo ""
echo "Test 2: CLI version"
VERSION=$(./dist/aithub-darwin-arm64 --version 2>&1 | grep -o "3.0.0" || true)
if [ "$VERSION" = "3.0.0" ]; then
  pass "CLI version correct (3.0.0)"
else
  fail "CLI version incorrect"
fi

# Test 3: CLI commands exist
echo ""
echo "Test 3: CLI commands"
COMMANDS=("search" "install" "rate" "submit" "status" "fork" "details")
for cmd in "${COMMANDS[@]}"; do
  if ./dist/aithub-darwin-arm64 "$cmd" --help &>/dev/null; then
    pass "Command '$cmd' exists"
  else
    fail "Command '$cmd' missing"
  fi
done

# Test 4: Discovery Skill format
echo ""
echo "Test 4: Discovery Skill format"
if [ -f "./skills/skillhub/SKILL.md" ]; then
  if grep -q "aithub search" "./skills/skillhub/SKILL.md"; then
    pass "Discovery Skill uses CLI commands"
  else
    fail "Discovery Skill doesn't reference CLI"
  fi
else
  fail "Discovery Skill not found"
fi

# Test 5: Routing rules exist
echo ""
echo "Test 5: Routing rules"
if [ -f "./scripts/skillhub-routing.md" ]; then
  if grep -q "aithub search" "./scripts/skillhub-routing.md"; then
    pass "Routing rules reference CLI"
  else
    fail "Routing rules don't reference CLI"
  fi
else
  fail "Routing rules not found"
fi

# Test 6: Install script updated
echo ""
echo "Test 6: Install scripts"
if grep -q "aithub CLI" "./scripts/install.sh"; then
  pass "install.sh includes CLI installation"
else
  fail "install.sh missing CLI installation"
fi

if grep -q "aithub CLI" "./scripts/install.ps1"; then
  pass "install.ps1 includes CLI installation"
else
  fail "install.ps1 missing CLI installation"
fi

# Test 7: Download handler exists
echo ""
echo "Test 7: Backend download handler"
if [ -f "./internal/handler/downloads.go" ]; then
  if grep -q "ServeDownload" "./internal/handler/downloads.go"; then
    pass "Download handler implemented"
  else
    fail "Download handler incomplete"
  fi
else
  fail "Download handler not found"
fi

# Test 8: API builds
echo ""
echo "Test 8: API compilation"
if go build -o /tmp/test-api ./cmd/api 2>/dev/null; then
  pass "API builds successfully"
  rm -f /tmp/test-api
else
  fail "API build failed"
fi

# Test 9: CLI help output
echo ""
echo "Test 9: CLI help output"
HELP_OUTPUT=$(./dist/aithub-darwin-arm64 --help 2>&1)
if echo "$HELP_OUTPUT" | grep -q "SkillHub CLI"; then
  pass "CLI help output correct"
else
  fail "CLI help output incorrect"
fi

# Test 10: All binaries built
echo ""
echo "Test 10: Multi-platform binaries"
EXPECTED_BINARIES=(
  "aithub-linux-amd64"
  "aithub-linux-arm64"
  "aithub-darwin-amd64"
  "aithub-darwin-arm64"
  "aithub-windows-amd64.exe"
)

for binary in "${EXPECTED_BINARIES[@]}"; do
  if [ -f "./dist/$binary" ]; then
    pass "Binary $binary exists"
  else
    warn "Binary $binary missing (run ./scripts/build_cli.sh)"
  fi
done

echo ""
echo "╔══════════════════════════════════════╗"
echo "║  ✓ All tests passed!                ║"
echo "╚══════════════════════════════════════╝"
echo ""
echo "System ready for deployment."
echo ""
echo "Next steps:"
echo "  1. Deploy CLI binaries to server: scp dist/* server:/var/www/skillhub/downloads/"
echo "  2. Deploy API with download handler"
echo "  3. Test install script: curl -fsSL https://your-domain.com/install | bash"
echo "  4. Verify AI auto-discovery works"
