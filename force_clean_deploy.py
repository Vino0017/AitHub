#!/usr/bin/env python3
"""强制清理并重新部署"""

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
    err = stderr.read().decode('utf-8', errors='ignore')
    if out.strip():
        for l in out.strip().split("\n")[:20]:
            print(f"  {l}")
    if err.strip():
        for l in err.strip().split("\n")[:10]:
            print(f"  [ERR] {l}")
    return out, err

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Force Clean Deploy ===\n")

# 1. 停止并删除所有容器
print("[1] Stopping and removing containers...")
run(client, f"cd {REMOTE_DIR} && docker compose down -v")

# 2. 删除所有相关镜像
print("\n[2] Removing all images...")
run(client, "docker rmi -f skillhub-api skillhub-postgres 2>/dev/null || true")
run(client, "docker system prune -af")

# 3. 清理构建缓存
print("\n[3] Cleaning build cache...")
run(client, "docker builder prune -af")

# 4. 验证新代码文件存在
print("\n[4] Verifying new code files...")
out, _ = run(client, f"ls -la {REMOTE_DIR}/internal/handler/bootstrap.go")
if "bootstrap.go" not in out:
    print("ERROR: bootstrap.go not found!")
    client.close()
    exit(1)

# 5. 测试编译
print("\n[5] Testing compilation...")
out, err = run(client, f"""
cd {REMOTE_DIR}
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c '
  go mod download &&
  go build -v -o skillhub ./cmd/api 2>&1 &&
  ls -lh skillhub
'
""", timeout=300)

if "skillhub" not in out and "skillhub" not in err:
    print("\nERROR: Build failed!")
    print(f"Output: {out}")
    print(f"Error: {err}")
    client.close()
    exit(1)

print("\n  BUILD SUCCESS!")

# 6. 重新构建镜像（无缓存）
print("\n[6] Building new image (no cache)...")
run(client, f"cd {REMOTE_DIR} && docker compose build --no-cache --progress=plain api", timeout=600)

# 7. 启动服务
print("\n[7] Starting services...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

# 8. 等待启动
print("\n[8] Waiting for startup...")
time.sleep(20)

# 9. 检查容器状态
print("\n[9] Checking container status...")
run(client, "docker ps")
run(client, f"docker logs skillhub-api-1 --tail 30")

# 10. 测试健康检查
print("\n[10] Testing health...")
for i in range(10):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n=== HEALTH CHECK PASSED! ===")
        break
    time.sleep(3)

# 11. 测试Bootstrap端点
print("\n[11] Testing Bootstrap endpoints...")
out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery")
if "discovery_skill" in out or "namespace" in out:
    print("\n=== BOOTSTRAP DISCOVERY WORKS! ===")
else:
    print("\n=== BOOTSTRAP DISCOVERY FAILED (404?) ===")
    print(out[:200])

out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/check")
print(f"\nBootstrap check: {out[:200]}")

# 12. 检查容器内文件
print("\n[12] Checking files in container...")
run(client, "docker exec skillhub-api-1 ls -la /app/internal/handler/ | grep bootstrap")

client.close()
print("\n=== DEPLOY COMPLETE ===")
print("URL: https://skillhub.koolkassanmsk.top")
