#!/usr/bin/env python3
"""紧急恢复 - 启动旧版本容器"""

import paramiko

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"

def run(client, cmd, timeout=120):
    print(f"$ {cmd[:80]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    if out.strip():
        for line in out.strip().split('\n')[:15]:
            print(f"  {line}")
    return out

print("=== Emergency Recovery ===\n")

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

# 检查容器状态
print("[1] Checking containers...")
run(client, "cd /opt/skillhub && docker compose ps -a")

# 启动已有的容器（不重新构建）
print("\n[2] Starting existing containers...")
run(client, "cd /opt/skillhub && docker compose up -d postgres api")

# 如果失败，尝试使用旧镜像
print("\n[3] Checking if service is up...")
import time
time.sleep(5)

out = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'down'")

if '"ok":true' not in out:
    print("\n[4] Service still down, checking logs...")
    run(client, "cd /opt/skillhub && docker compose logs api --tail 30")

    print("\n[5] Trying to restart with old image...")
    run(client, "cd /opt/skillhub && docker compose restart api")
    time.sleep(5)
    run(client, "curl -s http://localhost:8080/health")
else:
    print("\n=== Service is UP! ===")

client.close()
