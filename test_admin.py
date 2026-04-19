#!/usr/bin/env python3
"""测试管理员功能"""

import paramiko, json

HOST, USER, PASSWORD = "192.227.235.131", "root", "Vino"
ADMIN_TOKEN = "change-me-in-production"  # Default from deploy.sh

def run(client, cmd):
    _, stdout, stderr = client.exec_command(cmd, timeout=60)
    return stdout.read().decode('utf-8', errors='ignore')

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Admin Features Test ===\n")

# Test privacy scan with admin token
print("[1] Privacy Scan (Admin)")
out = run(client, f'curl -s -H "Authorization: Bearer {ADMIN_TOKEN}" http://localhost:8080/v1/admin/privacy/scan')
print(f"  Response: {out[:200]}")

# Test pending skills
print("\n[2] Pending Skills (Admin)")
out = run(client, f'curl -s -H "Authorization: Bearer {ADMIN_TOKEN}" http://localhost:8080/v1/admin/skills/pending')
try:
    data = json.loads(out)
    print(f"  OK Pending: {len(data.get('skills', []))}")
except:
    print(f"  Response: {out[:150]}")

# Check environment variables
print("\n[3] Environment Check")
out = run(client, "docker exec skillhub-api-1 env | grep -E 'ADMIN_TOKEN|LLM_API_KEY|DATABASE_URL'")
print(f"  Environment vars:")
for line in out.strip().split("\n"):
    if "=" in line:
        key, val = line.split("=", 1)
        if "TOKEN" in key or "KEY" in key:
            print(f"    {key}={val[:20]}...")
        else:
            print(f"    {key}={val}")

client.close()
print("\n=== Test Complete ===")
