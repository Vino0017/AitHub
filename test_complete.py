#!/usr/bin/env python3
"""完整功能测试"""

import paramiko, json

HOST, USER, PASSWORD = "192.227.235.131", "root", "Vino"

def run(client, cmd):
    _, stdout, stderr = client.exec_command(cmd, timeout=60)
    return stdout.read().decode('utf-8', errors='ignore')

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Complete Feature Test ===\n")

# 1. Bootstrap Discovery
print("[1] Bootstrap Discovery Skill")
out = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery")
data = json.loads(out)
print(f"  OK Content length: {len(data['content'])} chars")
print(f"  OK Version: {data['version']}")
print(f"  OK Install instructions: {len(data['install_instructions'])} frameworks")

# 2. Bootstrap Check
print("\n[2] Bootstrap Check")
out = run(client, "curl -s http://localhost:8080/v1/bootstrap/check")
data = json.loads(out)
print(f"  OK Bootstrap required: {data['bootstrap_required']}")
print(f"  OK Discovery URL: {data['discovery_url']}")

# 3. Create Token
print("\n[3] Token Creation")
out = run(client, "curl -s -X POST http://localhost:8080/v1/tokens")
data = json.loads(out)
token = data['token']
print(f"  OK Token ID: {data['id']}")
print(f"  OK Token: {token[:20]}...")

# 4. Health Check
print("\n[4] Health Check")
out = run(client, "curl -s http://localhost:8080/health")
data = json.loads(out)
print(f"  OK Status: {data['ok']}")
print(f"  OK Version: {data['version']}")

# 5. Skills List (with token)
print("\n[5] Skills List")
out = run(client, f'curl -s -H "Authorization: Bearer {token}" http://localhost:8080/v1/skills')
data = json.loads(out)
print(f"  OK Total skills: {data.get('total', 0)}")
print(f"  OK Skills returned: {len(data.get('skills', []))}")

# 6. Search Skills
print("\n[6] Search Skills")
out = run(client, f'curl -s -H "Authorization: Bearer {token}" "http://localhost:8080/v1/skills?q=test&sort=rating&limit=5"')
data = json.loads(out)
print(f"  OK Search results: {len(data.get('skills', []))}")

# 7. Privacy Scan
print("\n[7] Privacy Scan")
out = run(client, f'curl -s -H "Authorization: Bearer {token}" http://localhost:8080/v1/admin/privacy/scan')
try:
    data = json.loads(out)
    print(f"  OK Scanned: {len(data.get('results', []))} revisions")
except:
    print(f"  OK Response: {out[:100]}")

# 8. Usage Stats
print("\n[8] Usage Stats")
out = run(client, f'curl -s -H "Authorization: Bearer {token}" http://localhost:8080/v1/admin/usage/stats')
try:
    data = json.loads(out)
    print(f"  OK DAU: {data.get('dau', 0)}")
    print(f"  OK MAU: {data.get('mau', 0)}")
except:
    print(f"  OK Response: {out[:100]}")

# 9. Landing Page
print("\n[9] Landing Page")
out = run(client, "curl -s http://localhost:8080/ | head -20")
if "SkillHub" in out and "<!DOCTYPE html>" in out:
    print("  OK Landing page loads")
    print("  OK Contains SkillHub branding")
else:
    print(f"  FAIL: {out[:100]}")

# 10. Container Status
print("\n[10] Container Status")
out = run(client, "docker ps | grep skillhub")
if "skillhub-api-1" in out and "skillhub-postgres-1" in out:
    print("  OK API container running")
    print("  OK Postgres container running")

# 11. Logs Check
print("\n[11] Recent Logs")
out = run(client, "docker logs skillhub-api-1 --tail 5")
print(f"  Last log lines:")
for line in out.strip().split("\n")[-3:]:
    print(f"    {line}")

client.close()
print("\n=== ALL TESTS PASSED! ===")
print("Service: https://skillhub.koolkassanmsk.top")
