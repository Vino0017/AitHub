#!/usr/bin/env python3
"""快速修复并重启服务"""

import paramiko
import time

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"

def run(client, cmd, timeout=120):
    print(f"$ {cmd[:80]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')
    rc = stdout.channel.recv_exit_status()
    if out.strip():
        for line in out.strip().split('\n')[:15]:
            print(f"  {line}")
    if err.strip() and rc != 0:
        for line in err.strip().split('\n')[:8]:
            print(f"  [ERR] {line}")
    return out, rc

print("=== Quick Fix & Restart ===\n")

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

# 只构建API，不构建web
print("[1] Building API only...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d --build api", timeout=300)

print("\n[2] Waiting for service...")
time.sleep(10)

for i in range(20):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n=== API IS LIVE! ===")
        break
    time.sleep(2)

print("\n[3] Testing new features...")
run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -15")
run(client, "curl -s http://localhost:8080/v1/bootstrap/check")

client.close()
print("\nDone!")
