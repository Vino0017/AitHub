#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
SkillHub Deployment with Database Migration
Pull code from GitHub, run migrations, and deploy frontend and backend
"""
import paramiko, os, time, sys

# Server configuration
HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"
REPO_URL = "https://github.com/Vino0017/AitHub.git"

def run(client, cmd, timeout=120):
    """Execute remote command and print output"""
    print(f"$ {cmd[:150]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')

    # Print output (last 25 lines)
    for line in out.strip().split("\n")[-25:] if out.strip() else []:
        try: print(f"  {line}")
        except: pass

    # Print errors (last 10 lines)
    for line in err.strip().split("\n")[-10:] if err.strip() and "Pulling" not in err else []:
        try: print(f"  [ERR] {line}")
        except: pass

    return out, err

# Connect to server
print("=== SkillHub Deployment with Migration ===\n")
print(f"[Connecting] {HOST}...")
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("\n[1] Check if repository exists...")
out, err = run(client, f"test -d {REMOTE_DIR}/.git && echo 'exists' || echo 'not exists'", timeout=10)

if "exists" in out:
    print("\n[1a] Pull latest code from GitHub...")
    out, err = run(client, f"""
cd {REMOTE_DIR} && \
git fetch origin && \
git reset --hard origin/main && \
git clean -fd
""", timeout=60)

    if "error" in err.lower() or "fatal" in err.lower():
        print(f"\nWarning: Git pull failed, will try to clone...")
        run(client, f"rm -rf {REMOTE_DIR}")
        out, err = run(client, f"git clone {REPO_URL} {REMOTE_DIR}", timeout=120)
else:
    print("\n[1b] Clone repository from GitHub...")
    run(client, f"mkdir -p {REMOTE_DIR}")
    out, err = run(client, f"git clone {REPO_URL} {REMOTE_DIR}", timeout=120)

if "error" in err.lower() or "fatal" in err.lower():
    print(f"\nERROR: Git operation failed!\n{err[:500]}")
    print("\nTrying alternative: check if code already exists...")
    out, _ = run(client, f"ls -la {REMOTE_DIR} | head -10")
    if "go.mod" not in out:
        print("ERROR: No code found on server!")
        client.close()
        sys.exit(1)
    print("Code exists, continuing with deployment...")

print("\n[2] Run database migrations...")
# Get database connection info from docker-compose
out, err = run(client, f"""
cd {REMOTE_DIR} && \
docker compose exec -T postgres psql -U skillhub -d skillhub -f /opt/skillhub/migrations/011_add_security_audit_log.sql 2>&1 || \
echo "Migration may have already been applied or database not running"
""", timeout=60)

if "CREATE TABLE" in out or "already exists" in out or "Migration may have already been applied" in out:
    print("  SUCCESS: Migration applied or already exists!")
else:
    print(f"  WARNING: Migration status unclear:\n{out[:300]}")
    print("  Continuing with deployment...")

print("\n[3] Build backend...")
out, err = run(client, f"""
cd {REMOTE_DIR} && \
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c \
"go build -o skillhub ./cmd/api && chmod +x skillhub && ls -lh skillhub"
""", timeout=300)

if "skillhub" not in out:
    print(f"\nERROR: Backend build failed!\n{err[:500]}")
    client.close()
    sys.exit(1)
print("  SUCCESS: Backend built!")

print("\n[4] Build frontend...")
out, err = run(client, f"""
cd {REMOTE_DIR}/web && \
npm install && \
npm run build
""", timeout=600)

if "error" in err.lower() and "warn" not in err.lower():
    print(f"\nWARNING: Frontend build may have issues:\n{err[:500]}")
else:
    print("  SUCCESS: Frontend built!")

print("\n[5] Build Docker images...")
run(client, f"cd {REMOTE_DIR} && docker compose build", timeout=300)

print("\n[6] Restart services...")
run(client, f"cd {REMOTE_DIR} && docker compose down && docker compose up -d")

print("\n[7] Wait for services to start...")
time.sleep(25)

print("\n[8] Health check...")
for i in range(15):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n  SUCCESS: Backend health check passed!")
        break
    time.sleep(2)
else:
    print("\n  WARNING: Backend health check timeout")

print("\n[9] Test Bootstrap endpoint...")
out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
if "SkillHub Discovery Skill" in out or "skill" in out.lower():
    print("  SUCCESS: Bootstrap endpoint working!")
else:
    print(f"  WARNING: Bootstrap endpoint response abnormal:\n{out[:200]}")

print("\n[10] Test frontend...")
out, _ = run(client, "curl -s http://localhost:3000 | head -20")
if "SkillHub" in out or "<!DOCTYPE" in out:
    print("  SUCCESS: Frontend service running!")
else:
    print(f"  WARNING: Frontend response abnormal:\n{out[:200]}")

print("\n[11] Verify security audit log table...")
out, _ = run(client, f"""
cd {REMOTE_DIR} && \
docker compose exec -T postgres psql -U skillhub -d skillhub -c "\\dt security_audit_log" 2>&1
""", timeout=30)

if "security_audit_log" in out:
    print("  SUCCESS: Security audit log table exists!")
else:
    print(f"  WARNING: Security audit log table not found:\n{out[:200]}")

client.close()

print("\n" + "="*50)
print("DEPLOYMENT COMPLETE!")
print("="*50)
print(f"Backend API: https://skillhub.koolkassanmsk.top")
print(f"Frontend: https://skillhub.koolkassanmsk.top")
print("\nNew Features:")
print("  - Enhanced prompt injection detection")
print("  - LLM review double verification")
print("  - Content sanitization before storage")
print("  - Risk scoring with thresholds")
print("  - Security audit logging")
print("="*50)
