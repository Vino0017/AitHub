"""
SkillHub AI Review Pipeline Test.
Reads credentials from server.MD (gitignored).
"""
import paramiko, time, json, hashlib, secrets, urllib.request, urllib.error, os, sys

def load_server_config():
    config_path = os.path.join(os.path.dirname(__file__), "..", "server.MD")
    if not os.path.exists(config_path):
        sys.exit("ERROR: server.MD not found.")
    config = {}
    with open(config_path, "r", encoding="utf-8") as f:
        for line in f:
            if line.startswith("ssh root@"):
                config["host"] = line.split("@")[1].strip()
            if "密码" in line and ("：" in line or ":" in line):
                config["password"] = line.split("：" if "：" in line else ":")[1].strip()
            if "域名" in line and ("：" in line or ":" in line):
                config["domain"] = line.split("：" if "：" in line else ":")[1].strip()
    return config

cfg = load_server_config()
HOST = cfg["host"]
API = f"https://{cfg['domain']}"

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(HOST, username="root", password=cfg["password"], timeout=15)

def ssh(cmd, timeout=60):
    _, stdout, _ = c.exec_command(cmd, timeout=timeout)
    return stdout.read().decode().strip()

# 1. Create test user + token
raw = "sk_" + secrets.token_hex(32)
h = hashlib.sha256(raw.encode()).hexdigest()

sftp = c.open_sftp()
with sftp.open("/opt/skillhub/setup.sql", "w") as f:
    f.write(f"INSERT INTO namespaces (name, type, email) VALUES ('tester', 'personal', 'tester@t.com') ON CONFLICT DO NOTHING;\n")
    f.write(f"INSERT INTO tokens (namespace_id, token_hash, label) SELECT id, '{h}', 'apitest' FROM namespaces WHERE name = 'tester';\n")
sftp.close()
ssh("docker cp /opt/skillhub/setup.sql skillhub-postgres-1:/tmp/s.sql && cd /opt/skillhub && docker compose exec -T postgres psql -U skillhub -d skillhub -f /tmp/s.sql")
print(f"Token: {raw[:25]}...")

# 2. Submit skill
print("\n=== Submit skill ===")
content = """---
name: greet-helper
version: 1.0.0
schema: skill-md
framework: gstack
tags: [greeting, helper]
description: "Helps agents greet users in different languages"
triggers: [greet, hello]
compatible_models: [claude-3-5-sonnet, gpt-4o]
---

# Greet Helper
When the user wants to greet someone, provide a culturally appropriate greeting.
"""

data = json.dumps({"content": content}).encode("utf-8")
req = urllib.request.Request(f"{API}/v1/skills", data=data,
    headers={"Authorization": f"Bearer {raw}", "Content-Type": "application/json"}, method="POST")
try:
    with urllib.request.urlopen(req, timeout=15) as resp:
        print(f"  {resp.status}: {resp.read().decode()}")
except urllib.error.HTTPError as e:
    print(f"  HTTP {e.code}: {e.read().decode()}")

# 3. Wait + check review
print("\n=== Wait 10s for AI review ===")
time.sleep(10)
status = ssh("cd /opt/skillhub && docker compose exec -T postgres psql -U skillhub -d skillhub -t -A -c \"SELECT review_status FROM revisions ORDER BY created_at DESC LIMIT 1\"")
print(f"  Review status: {status}")

logs = ssh("cd /opt/skillhub && docker compose logs api --tail 5 2>&1")
for l in logs.split("\n")[-5:]:
    print(f"  {l}")

# 4. Search
print("\n=== Search ===")
req2 = urllib.request.Request(f"{API}/v1/skills?q=greet", headers={"Authorization": f"Bearer {raw}"})
try:
    with urllib.request.urlopen(req2, timeout=10) as resp:
        result = json.loads(resp.read().decode())
        print(f"  Found {result['total']} skills")
except Exception as e:
    print(f"  {e}")

c.close()
print("\nDone!")
