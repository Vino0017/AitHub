#!/usr/bin/env python3
import paramiko, os, time, sys

HOST, USER, PASSWORD, REMOTE_DIR = "192.227.235.131", "root", "Vino", "/opt/skillhub"
PROJECT_DIR = os.getcwd()

def run(client, cmd, timeout=120):
    print(f"$ {cmd[:150]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out, err = stdout.read().decode('utf-8', errors='ignore'), stderr.read().decode('utf-8', errors='ignore')
    for l in out.strip().split("\n")[-25:] if out.strip() else []:
        try: print(f"  {l}")
        except: pass
    for l in err.strip().split("\n")[-10:] if err.strip() and "Pulling" not in err else []:
        try: print(f"  [ERR] {l}")
        except: pass
    return out, err

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)
sftp = client.open_sftp()

print("=== Final Deploy ===\n")

print("[1] Uploading...")
for f in ["internal/handler/admin.go", "internal/handler/bootstrap.go", "internal/handler/web.go", "internal/handler/rating.go", "internal/handler/skill_detail.go"]:
    sftp.put(os.path.join(PROJECT_DIR, f.replace("/", os.sep)), f"{REMOTE_DIR}/{f}")
    print(f"  + {f}")

print("\n[2] Building...")
out, err = run(client, f'cd {REMOTE_DIR} && docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c "go build -o skillhub ./cmd/api && chmod +x skillhub && ls -lh skillhub"', timeout=300)
if "skillhub" not in out:
    print(f"\nBUILD FAILED!\n{err[:500]}")
    sftp.close(); client.close(); sys.exit(1)
print("\n  BUILD OK!")

print("\n[3] Building image...")
run(client, f"cd {REMOTE_DIR} && docker compose build api", timeout=300)

print("\n[4] Starting...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

print("\n[5] Waiting...")
time.sleep(25)

print("\n[6] Testing...")
for i in range(15):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n=== HEALTH OK! ===")
        break
    time.sleep(2)

print("\n[7] Bootstrap...")
out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
print("\n=== BOOTSTRAP WORKS! ===" if "SkillHub Discovery Skill" in out else f"\n=== FAILED ===\n{out[:200]}")

out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/check")
print(f"\nCheck: {out[:100]}")

print("\n[8] Features...")
run(client, "curl -s -X POST http://localhost:8080/v1/tokens | head -3")
run(client, "curl -s http://localhost:8080/v1/skills | head -5")

sftp.close(); client.close()
print("\n=== DEPLOYED! ===\nURL: https://skillhub.koolkassanmsk.top")
