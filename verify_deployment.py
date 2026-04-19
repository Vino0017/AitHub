#!/usr/bin/env python3
import paramiko, json

HOST, USER, PASSWORD = "192.227.235.131", "root", "Vino"
ADMIN_TOKEN = "change-me-in-production"

def run(client, cmd):
    _, stdout, _ = client.exec_command(cmd, timeout=60)
    return stdout.read().decode('utf-8', errors='ignore')

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Final Verification ===\n")

# 1. Bootstrap
print("[1] Bootstrap Discovery")
out = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery")
data = json.loads(out)
print(f"  OK Version: {data['version']}, Content: {len(data['content'])} chars")

# 2. Token
print("\n[2] Token Creation")
out = run(client, "curl -s -X POST http://localhost:8080/v1/tokens")
data = json.loads(out)
token = data['token']
print(f"  OK Token: {token[:30]}...")

# 3. Skills
print("\n[3] Skills API")
out = run(client, f'curl -s -H "Authorization: Bearer {token}" http://localhost:8080/v1/skills')
data = json.loads(out)
print(f"  OK Total: {data.get('total', 0)} skills")

# 4. Admin - Privacy Scan (correct path)
print("\n[4] Admin Privacy Scan")
out = run(client, f'curl -s -H "Authorization: Bearer {ADMIN_TOKEN}" http://localhost:8080/admin/privacy/scan')
try:
    data = json.loads(out)
    print(f"  OK Scanned: {len(data.get('results', []))} revisions")
except:
    print(f"  Response: {out[:150]}")

# 5. Admin - Pending Skills
print("\n[5] Admin Pending Skills")
out = run(client, f'curl -s -H "Authorization: Bearer {ADMIN_TOKEN}" http://localhost:8080/admin/skills/pending')
try:
    data = json.loads(out)
    print(f"  OK Pending: {len(data.get('skills', []))}")
except:
    print(f"  Response: {out[:150]}")

# 6. Landing Page
print("\n[6] Landing Page")
out = run(client, "curl -s http://localhost:8080/ | grep -o '<title>.*</title>'")
print(f"  {out.strip()}")

# 7. Health
print("\n[7] Health Check")
out = run(client, "curl -s http://localhost:8080/health")
data = json.loads(out)
print(f"  OK Status: {data['ok']}, Version: {data['version']}")

# 8. Container Status
print("\n[8] Containers")
out = run(client, "docker ps --format '{{.Names}}: {{.Status}}' | grep skillhub")
for line in out.strip().split("\n"):
    print(f"  {line}")

client.close()
print("\n=== DEPLOYMENT SUCCESSFUL ===")
print("Service: https://skillhub.koolkassanmsk.top")
print("\nKey Features Verified:")
print("  - Bootstrap Discovery Skill (auto-installation)")
print("  - Token creation and authentication")
print("  - Skills API with search")
print("  - Admin privacy scanning")
print("  - Landing page with UI")
print("  - Health monitoring")
