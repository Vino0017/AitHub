#!/usr/bin/env python3
"""完整重建流程"""

import paramiko
import time

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"

def run(client, cmd, timeout=120):
    print(f"$ {cmd[:150]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')
    if out.strip():
        lines = out.strip().split("\n")
        for l in lines[-30:]:
            print(f"  {l}")
    if err.strip() and "Pulling" not in err:
        lines = err.strip().split("\n")
        for l in lines[-15:]:
            print(f"  [ERR] {l}")
    return out, err

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Complete Rebuild ===\n")

# 1. 构建二进制
print("[1] Building binary...")
out, err = run(client, f"""
cd {REMOTE_DIR}
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c '
  go build -o skillhub ./cmd/api &&
  chmod +x skillhub &&
  ls -lh skillhub
'
""", timeout=300)

if "skillhub" not in out:
    print("\nERROR: Binary build failed!")
    print(err)
    client.close()
    exit(1)

print("\n  Binary built successfully!")

# 2. 验证二进制
print("\n[2] Verifying binary...")
run(client, f"ls -lh {REMOTE_DIR}/skillhub")

# 3. 构建Docker镜像
print("\n[3] Building Docker image...")
run(client, f"cd {REMOTE_DIR} && docker compose build api", timeout=300)

# 4. 启动服务
print("\n[4] Starting services...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

# 5. 等待
print("\n[5] Waiting for startup...")
time.sleep(25)

# 6. 检查状态
print("\n[6] Checking status...")
run(client, "docker ps | grep skillhub")
run(client, f"docker logs skillhub-api-1 --tail 20")

# 7. 测试健康
print("\n[7] Testing health...")
for i in range(15):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n=== HEALTH CHECK PASSED! ===")
        break
    time.sleep(2)

# 8. 测试Bootstrap
print("\n[8] Testing Bootstrap...")
out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery")
if "discovery_skill" in out or "namespace" in out:
    print("\n=== BOOTSTRAP WORKS! ===")
    print(out[:300])
else:
    print("\n=== BOOTSTRAP FAILED ===")
    print(out[:200])

out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/check")
print(f"\nBootstrap check: {out[:200]}")

# 9. 测试其他新功能
print("\n[9] Testing other features...")
run(client, "curl -s http://localhost:8080/v1/skills | head -20")

client.close()
print("\n=== DONE ===")
print("URL: https://skillhub.koolkassanmsk.top")
