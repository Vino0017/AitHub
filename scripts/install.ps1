# SkillHub Installer for Windows (PowerShell)
# Usage:
#   irm https://skillhub.koolkassanmsk.top/install.ps1 | iex
#   irm https://skillhub.koolkassanmsk.top/install.ps1 | iex -register -github

param(
    [switch]$register,
    [switch]$github,
    [switch]$google,
    [string]$email,
    [string]$namespace,
    [string]$api = "https://skillhub.koolkassanmsk.top"
)

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "╔══════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║        SkillHub Installer v2         ║" -ForegroundColor Cyan
Write-Host "║   AI-First Skill Registry           ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Detect OS
$os = "windows"
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
Write-Host "→ OS: $os ($arch)" -ForegroundColor Gray

# Registration
$token = $env:SKILLHUB_TOKEN

if ($register) {
    Write-Host ""
    Write-Host "── Registration ──" -ForegroundColor Yellow

    if ($github) {
        Write-Host "→ Starting GitHub OAuth Device Flow..." -ForegroundColor White
        $deviceResp = Invoke-RestMethod -Uri "$api/v1/auth/github" -Method POST -ContentType "application/json"

        Write-Host ""
        Write-Host "  ┌────────────────────────────────────┐" -ForegroundColor Green
        Write-Host "  │ Open: $($deviceResp.verification_uri)" -ForegroundColor Green
        Write-Host "  │ Enter code: $($deviceResp.user_code)" -ForegroundColor Green
        Write-Host "  └────────────────────────────────────┘" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Waiting for authorization..." -ForegroundColor Gray

        for ($i = 0; $i -lt 60; $i++) {
            Start-Sleep -Seconds 5
            try {
                $pollResp = Invoke-RestMethod -Uri "$api/v1/auth/github/poll" -Method POST `
                    -ContentType "application/json" `
                    -Body (@{device_code = $deviceResp.device_code} | ConvertTo-Json)
                if ($pollResp.status -eq "complete") {
                    $token = $pollResp.token
                    $ns = $pollResp.namespace
                    Write-Host "  ✓ Authorized! Namespace: $ns" -ForegroundColor Green
                    break
                }
            } catch {}
        }
    }
    elseif ($email) {
        Write-Host "→ Sending verification to $email..." -ForegroundColor White
        Invoke-RestMethod -Uri "$api/v1/auth/email/send" -Method POST `
            -ContentType "application/json" `
            -Body (@{email=$email; namespace=$namespace} | ConvertTo-Json) | Out-Null

        $code = Read-Host "  Enter verification code"
        $verifyResp = Invoke-RestMethod -Uri "$api/v1/auth/email/verify" -Method POST `
            -ContentType "application/json" `
            -Body (@{email=$email; code=$code} | ConvertTo-Json)
        $token = $verifyResp.token
        $ns = $verifyResp.namespace
        Write-Host "  ✓ Verified! Namespace: $ns" -ForegroundColor Green
    }
}

# Create anonymous token if none
if (-not $token) {
    Write-Host "→ Creating anonymous token..." -ForegroundColor Gray
    $tokenResp = Invoke-RestMethod -Uri "$api/v1/tokens" -Method POST
    $token = $tokenResp.token
    Write-Host "  ✓ Anonymous token created" -ForegroundColor Green
}

# Detect frameworks
Write-Host ""
Write-Host "── Detecting AI Agent Frameworks ──" -ForegroundColor Yellow

$frameworks = @{
    "gstack"      = "$env:USERPROFILE\.gstack\skills"
    "openclaw"    = "$env:USERPROFILE\.openclaw\skills"
    "hermes"      = "$env:USERPROFILE\.hermes\skills"
    "claude-code" = "$env:USERPROFILE\.claude\skills"
    "cursor"      = "$env:USERPROFILE\.cursor\skills"
    "windsurf"    = "$env:USERPROFILE\.windsurf\skills"
}

$installedCount = 0
$failedCount = 0
foreach ($fw in $frameworks.Keys) {
    $dir = $frameworks[$fw]
    $parentDir = Split-Path $dir -Parent
    if (Test-Path $parentDir) {
        Write-Host "  ✓ $fw detected → $dir" -ForegroundColor Green
        New-Item -ItemType Directory -Path "$dir\skillhub" -Force | Out-Null

        try {
            $content = Invoke-RestMethod -Uri "$api/v1/skills/skillhub-demo/skillhub-discovery/content" `
                -Headers @{Authorization="Bearer $token"}
            if ($content.content) {
                $content.content | Out-File -FilePath "$dir\skillhub\SKILL.md" -Encoding UTF8
                $installedCount++
            } else {
                Write-Host "    ⚠ Empty content returned for $fw" -ForegroundColor Yellow
                $failedCount++
            }
        } catch {
            Write-Host "    ⚠ Failed to install discovery skill for $fw`: $_" -ForegroundColor Yellow
            $failedCount++
        }
    }
}

if ($installedCount -eq 0) {
    Write-Host "  ⚠ No AI agent frameworks detected" -ForegroundColor Yellow
}

# Environment variable
Write-Host ""
Write-Host "── Configuring Environment ──" -ForegroundColor Yellow
[Environment]::SetEnvironmentVariable("SKILLHUB_TOKEN", $token, "User")
$env:SKILLHUB_TOKEN = $token
Write-Host "  ✓ SKILLHUB_TOKEN set as user environment variable" -ForegroundColor Green

# PowerShell profile
$profilePath = $PROFILE.CurrentUserAllHosts
if (-not (Test-Path $profilePath)) {
    New-Item -ItemType File -Path $profilePath -Force | Out-Null
}
$profileContent = Get-Content $profilePath -Raw -ErrorAction SilentlyContinue
if ($profileContent -notmatch "SKILLHUB_TOKEN") {
    Add-Content $profilePath "`n`$env:SKILLHUB_TOKEN = `"$token`""
    Write-Host "  ✓ Token added to PowerShell profile" -ForegroundColor Green
}

Write-Host ""
Write-Host "╔══════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║  ✓ SkillHub installed successfully!  ║" -ForegroundColor Green
Write-Host "╚══════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "Token: $($token.Substring(0,20))..."
if ($ns) { Write-Host "Namespace: $ns" }
Write-Host "Frameworks: $installedCount detected"
Write-Host ""
Write-Host "Your AI agents can now discover and use skills from SkillHub." -ForegroundColor White
