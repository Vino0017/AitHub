"""
SkillHub Quick Review Test.
Reads credentials from server.MD (gitignored).
"""
import paramiko, os, sys

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

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(cfg["host"], username="root", password=cfg["password"], timeout=15)

def run(cmd, timeout=60):
    print(f"$ {cmd[:120]}")
    _, stdout, stderr = c.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode()
    if out.strip():
        for l in out.strip().split("\n")[:10]: print(f"  {l}")
    return out

print("=== Health check ===")
run(f"curl -s https://{cfg['domain']}/health")

print("\n=== API logs ===")
run("cd /opt/skillhub && docker compose logs api --tail 10 2>&1")

c.close()
print("\nDone!")
