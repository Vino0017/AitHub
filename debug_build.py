#!/usr/bin/env python3
"""调试Docker构建问题"""

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
        print(f"STDERR: {err}")
    return out

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Debug Docker Build ===\n")

# 检查构建上下文
print("[1] Checking build context...")
run(client, f"cd {REMOTE_DIR} && ls -la cmd/api/")
run(client, f"cd {REMOTE_DIR} && cat go.mod | head -10")

# 尝试手动构建看详细错误
print("\n[2] Manual build test...")
run(client, f"cd {REMOTE_DIR} && docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c 'go mod download && go build -o skillhub ./cmd/api'", timeout=300)

client.close()
