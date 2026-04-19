#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
SkillHub Deployment Script
Upload local code and deploy frontend and backend
"""
import paramiko, os, time, sys, tarfile, io

# Server configuration
HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"
PROJECT_DIR = os.getcwd()

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
print("=== SkillHub Deployment ===\n")
print(f"[Connecting] {HOST}...")
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)
sftp = client.open_sftp()

print("\n[1] Creating tarball of project...")
tar_buffer = io.BytesIO()
with tarfile.open(fileobj=tar_buffer, mode='w:gz') as tar:
    # Add all files except .git, node_modules, .next, etc.
    exclude_dirs = {'.git', 'node_modules', '.next', '__pycache__', '.gstack', 'web/node_modules', 'web/.next'}

    for root, dirs, files in os.walk(PROJECT_DIR):
        # Remove excluded directories from dirs list
        dirs[:] = [d for d in dirs if d not in exclude_dirs and not d.startswith('.')]

        for file in files:
            if file.startswith('.') and file not in ['.env', '.env.example', '.dockerignore']:
                continue
            file_path = os.path.join(root, file)
            arcname = os.path.relpath(file_path, PROJECT_DIR)
            try:
                tar.add(file_path, arcname=arcname)
            except Exception as e:
                print(f"  Warning: Could not add {arcname}: {e}")

tar_buffer.seek(0)
print(f"  Tarball size: {len(tar_buffer.getvalue()) / 1024 / 1024:.2f} MB")

print("\n[2] Uploading code to server...")
remote_tar = f"{REMOTE_DIR}/project.tar.gz"
run(client, f"mkdir -p {REMOTE_DIR}")
sftp.putfo(tar_buffer, remote_tar)
print("  Upload complete!")

print("\n[3] Extracting code...")
run(client, f"cd {REMOTE_DIR} && tar -xzf project.tar.gz && rm project.tar.gz")

print("\n[4] Build backend...")
out, err = run(client, f"""
cd {REMOTE_DIR} && \
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c \
"go build -o skillhub ./cmd/api && chmod +x skillhub && ls -lh skillhub"
""", timeout=300)

if "skillhub" not in out:
    print(f"\nERROR: Backend build failed!\n{err[:500]}")
    sftp.close()
    client.close()
    sys.exit(1)
print("  SUCCESS: Backend built!")

print("\n[5] Build Docker images...")
run(client, f"cd {REMOTE_DIR} && docker compose build", timeout=600)

print("\n[6] Start services...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

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

sftp.close()
client.close()

print("\n" + "="*50)
print("DEPLOYMENT COMPLETE!")
print("="*50)
print(f"Backend API: https://skillhub.koolkassanmsk.top")
print(f"Frontend: https://skillhub.koolkassanmsk.top (if configured)")
print("="*50)
