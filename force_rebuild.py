#!/usr/bin/env python3
"""强制重新构建 - 确保使用新代码"""

import paramiko
import time

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"

def run(client, cmd, timeout=120):
    print(f"$ {cmd[:100]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    if out.strip():
        for l in out.strip().split("\n")[:20]:
            print(f"  {l}")
    return out

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Force Rebuild ===\n")

# 1. 停止所有容器
print("[1] Stopping containers...")
run(client, f"cd {REMOTE_DIR} && docker compose down")

# 2. 删除旧二进制
print("\n[2] Removing old binary...")
run(client, f"rm -f {REMOTE_DIR}/skillhub")

# 3. 重新构建
print("\n[3] Building new binary...")
run(client, f"""
cd {REMOTE_DIR}
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c '
  go build -o skillhub ./cmd/api &&
  chmod +x skillhub &&
  ls -lh skillhub &&
  ./skillhub --version 2>&1 || echo "Binary created"
'
""", timeout=300)

# 4. 删除旧镜像
print("\n[4] Removing old images...")
run(client, "docker rmi skillhub-api 2>/dev/null || echo 'No old image'")

# 5. 重新构建镜像
print("\n[5] Building new image...")
run(client, f"cd {REMOTE_DIR} && docker compose build --no-cache api", timeout=300)

# 6. 启动
print("\n[6] Starting...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

# 7. 等待
print("\n[7] Waiting...")
time.sleep(15)

# 8. 测试
print("\n[8] Testing...")
run(client, "curl -s http://localhost:8080/health")
run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
run(client, "curl -s http://localhost:8080/v1/bootstrap/check")

# 9. 检查容器内的文件
print("\n[9] Checking container...")
run(client, "docker exec skillhub-api-1 ls -la /app/")
run(client, "docker exec skillhub-api-1 ls -la /app/internal/handler/ | grep bootstrap")

client.close()
print("\n=== DONE ===")
