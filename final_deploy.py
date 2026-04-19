#!/usr/bin/env python3
"""上传修复后的文件并部署"""

import paramiko
import os
import time
import sys

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
            try:
                print(f"  {l}")
            except:
                print(f"  {l.encode('utf-8', errors='ignore').decode('utf-8', errors='ignore')}")
    if err.strip() and "Pulling" not in err and "Download" not in err:
        lines = err.strip().split("\n")
        for l in lines[-10:]:
            try:
                print(f"  [ERR] {l}")
            except:
                pass
    return out, err

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)
sftp = client.open_sftp()

print("=== Upload & Deploy ===\n")

# 1. 上传修复后的文件
print("[1] Uploading fixed files...")
files = [
    ("internal/handler/bootstrap.go", f"{REMOTE_DIR}/internal/handler/bootstrap.go"),
    ("internal/handler/web.go", f"{REMOTE_DIR}/internal/handler/web.go"),
]
for local_rel, remote in files:
    local = os.path.join(PROJECT_DIR, local_rel.replace("/", os.sep))
    sftp.put(local, remote)
    print(f"  + {local_rel}")

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
    print("\nBUILD FAILED!")
    print(err[:500])
    sftp.close()
    client.close()
    sys.exit(1)

print("\n  BUILD SUCCESS!")

# 3. 构建镜像
print("\n[3] Building image...")
run(client, f"cd {REMOTE_DIR} && docker compose build api", timeout=300)

# 4. 启动
print("\n[4] Starting...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

# 5. 等待
print("\n[5] Waiting...")
time.sleep(25)

# 6. 测试
print("\n[6] Testing...")
for i in range(15):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n=== HEALTH OK! ===")
        break
    time.sleep(2)

# 7. Bootstrap测试
print("\n[7] Bootstrap test...")
out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
if "SkillHub Discovery Skill" in out:
    print("\n=== BOOTSTRAP WORKS! ===")
else:
    print(f"\n=== BOOTSTRAP FAILED ===\n{out[:200]}")

out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/check")
print(f"\nCheck: {out[:100]}")

# 8. 其他功能
print("\n[8] Other features...")
run(client, "curl -s -X POST http://localhost:8080/v1/tokens | head -3")
run(client, "curl -s http://localhost:8080/v1/skills | head -5")

sftp.close()
client.close()
print("\n=== DEPLOYED! ===")
print("URL: https://skillhub.koolkassanmsk.top")
