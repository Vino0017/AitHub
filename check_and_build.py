#!/usr/bin/env python3
"""检查服务器代码并手动构建"""

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
        for line in out.strip().split('\n')[:20]:
            print(f"  {line}")
    if err.strip():
        for line in err.strip().split('\n')[:10]:
            print(f"  [ERR] {line}")
    return out

print("=== Check & Manual Build ===\n")

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

# 检查新文件是否存在
print("[1] Checking uploaded files...")
run(client, f"ls -la {REMOTE_DIR}/internal/handler/bootstrap.go")
run(client, f"ls -la {REMOTE_DIR}/internal/privacy/cleaner.go")
run(client, f"ls -la {REMOTE_DIR}/migrations/010_add_usage_stats.sql")

# 检查Docker Compose配置
print("\n[2] Checking docker-compose.yml...")
run(client, f"cat {REMOTE_DIR}/docker-compose.yml")

# 尝试在容器内构建
print("\n[3] Building inside container...")
run(client, f"""
cd {REMOTE_DIR}
docker compose exec -T api sh -c 'cd /app && go build -o skillhub ./cmd/api && ls -lh skillhub'
""", timeout=300)

# 如果失败，尝试重新构建镜像
print("\n[4] Rebuilding image with cache...")
run(client, f"cd {REMOTE_DIR} && docker compose build --no-cache api", timeout=600)

# 重启
print("\n[5] Restarting...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d api")

time.sleep(10)

# 测试
print("\n[6] Testing...")
run(client, "curl -s http://localhost:8080/health")
run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -15")

client.close()
print("\nDone!")
