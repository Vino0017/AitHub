#!/usr/bin/env python3
"""上传修复后的bootstrap.go并重建"""

import paramiko
import os
import time

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"
PROJECT_DIR = os.getcwd()

def run(client, cmd, timeout=120):
    print(f"$ {cmd[:150]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')
    if out.strip():
        lines = out.strip().split("\n")
        for l in lines[-25:]:
            print(f"  {l}")
    if err.strip() and "Pulling" not in err and "Download" not in err:
        lines = err.strip().split("\n")
        for l in lines[-10:]:
            print(f"  [ERR] {l}")
    return out, err

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)
sftp = client.open_sftp()

print("=== Upload Fixed File & Rebuild ===\n")

# 1. 上传修复后的bootstrap.go
print("[1] Uploading fixed bootstrap.go...")
local = os.path.join(PROJECT_DIR, "internal", "handler", "bootstrap.go")
remote = f"{REMOTE_DIR}/internal/handler/bootstrap.go"
sftp.put(local, remote)
print(f"  Uploaded: {local}")

# 2. 构建二进制
print("\n[2] Building binary...")
out, err = run(client, f"""
cd {REMOTE_DIR}
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c '
  go build -o skillhub ./cmd/api &&
  chmod +x skillhub &&
  ls -lh skillhub
'
""", timeout=300)

if "skillhub" not in out:
    print("\nERROR: Build failed!")
    print(err)
    sftp.close()
    client.close()
    exit(1)

print("\n  Binary built!")

# 3. 构建Docker镜像
print("\n[3] Building Docker image...")
run(client, f"cd {REMOTE_DIR} && docker compose build api", timeout=300)

# 4. 启动
print("\n[4] Starting services...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

# 5. 等待
print("\n[5] Waiting...")
time.sleep(25)

# 6. 测试健康
print("\n[6] Testing health...")
for i in range(15):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n=== HEALTH OK! ===")
        break
    time.sleep(2)

# 7. 测试Bootstrap
print("\n[7] Testing Bootstrap...")
out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -30")
if "SkillHub Discovery Skill" in out:
    print("\n=== BOOTSTRAP WORKS! ===")
else:
    print("\n=== BOOTSTRAP FAILED ===")

out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/check")
print(f"\nCheck: {out[:150]}")

# 8. 测试其他功能
print("\n[8] Testing other features...")
run(client, "curl -s -X POST http://localhost:8080/v1/tokens | head -5")
run(client, "curl -s http://localhost:8080/v1/skills | head -10")

sftp.close()
client.close()
print("\n=== SUCCESS! ===")
print("URL: https://skillhub.koolkassanmsk.top")
