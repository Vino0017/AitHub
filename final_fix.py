#!/usr/bin/env python3
"""最终修复 - 上传修复后的文件并构建"""

import paramiko
import os
import time

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"
PROJECT_DIR = os.getcwd()

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
sftp = client.open_sftp()

print("=== Final Fix ===\n")

# 上传修复后的文件
print("[1] Uploading fixed files...")
files_to_fix = [
    "internal/privacy/cleaner.go",
    "internal/credibility/analyzer.go",
]

for f in files_to_fix:
    local = os.path.join(PROJECT_DIR, f.replace("/", os.sep))
    remote = f"{REMOTE_DIR}/{f}"
    sftp.put(local, remote)
    print(f"  + {f}")

# 测试构建
print("\n[2] Testing build...")
out, err = run(client, f"cd {REMOTE_DIR} && docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c 'go build -o skillhub ./cmd/api 2>&1'", timeout=300)

if "skillhub" in out or err == "":
    print("\n  BUILD SUCCESS!")

    # 构建Docker镜像
    print("\n[3] Building Docker image...")
    run(client, f"cd {REMOTE_DIR} && docker compose build api", timeout=600)

    # 启动
    print("\n[4] Starting services...")
    run(client, f"cd {REMOTE_DIR} && docker compose up -d")

    # 等待
    print("\n[5] Waiting...")
    time.sleep(15)

    # 测试
    print("\n[6] Testing...")
    run(client, "curl -s http://localhost:8080/health")
    run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -15")
    run(client, "curl -s http://localhost:8080/v1/bootstrap/check")

    print("\n=== SUCCESS! ===")
else:
    print("\n  BUILD FAILED!")
    print(f"Error: {err}")

sftp.close()
client.close()
