#!/usr/bin/env python3
"""检查构建失败原因"""

import paramiko

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"

def run(client, cmd, timeout=120):
    print(f"$ {cmd}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')
    print(out)
    if err:
        print(f"[STDERR]\n{err}")
    return out, err

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Checking Build Failure ===\n")

# 检查docker compose状态
print("[1] Docker compose status:")
run(client, f"cd {REMOTE_DIR} && docker compose ps -a")

# 检查构建日志
print("\n[2] Last build logs:")
run(client, f"cd {REMOTE_DIR} && docker compose logs --tail 50")

# 检查skillhub二进制是否存在
print("\n[3] Check binary:")
run(client, f"ls -lh {REMOTE_DIR}/skillhub")

# 检查Dockerfile.simple
print("\n[4] Check Dockerfile.simple:")
run(client, f"cat {REMOTE_DIR}/Dockerfile.simple")

# 尝试手动启动看错误
print("\n[5] Try manual start:")
run(client, f"cd {REMOTE_DIR} && docker compose up --no-build 2>&1 | head -50")

client.close()
