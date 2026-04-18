"""Restart API + Postgres only (without web frontend)."""
import paramiko, os, sys, time

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
    return config

cfg = load_server_config()
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(cfg["host"], username="root", password=cfg["password"], timeout=15)

def run(cmd, timeout=120):
    print(f"$ {cmd[:150]}")
    _, stdout, stderr = c.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode()
    err = stderr.read().decode()
    if out.strip():
        for l in out.strip().split("\n")[:20]: print(f"  {l}")
    if err.strip():
        for l in err.strip().split("\n")[:20]: print(f"  [err] {l}")

# Only start postgres + api (skip web which hasn't been deployed yet)
print("=== Starting postgres + api only ===")
run("cd /opt/skillhub && docker compose up -d --build postgres api 2>&1", timeout=300)

print("\n=== Waiting 15s ===")
time.sleep(15)

run("cd /opt/skillhub && docker compose ps 2>&1")
run("cd /opt/skillhub && docker compose logs api --tail 20 2>&1")
run("curl -s http://localhost:8080/health 2>&1")

c.close()
