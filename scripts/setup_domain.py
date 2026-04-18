"""
SkillHub Domain + SSL Setup Script.

Reads credentials from server.MD (not committed to repo).
Usage: python scripts/setup_domain.py
"""
import paramiko, time, os, sys

def load_server_config():
    """Load server config from server.MD (gitignored)."""
    config_path = os.path.join(os.path.dirname(__file__), "..", "server.MD")
    if not os.path.exists(config_path):
        sys.exit("ERROR: server.MD not found. Create it with SSH credentials.")
    
    config = {}
    with open(config_path, "r", encoding="utf-8") as f:
        for line in f:
            if line.startswith("ssh root@"):
                config["host"] = line.split("@")[1].strip()
            if line.startswith("密码：") or line.startswith("密码:"):
                config["password"] = line.split("：" if "：" in line else ":")[1].strip()
            if line.startswith("域名：") or line.startswith("域名:"):
                config["domain"] = line.split("：" if "：" in line else ":")[1].strip()
    return config

cfg = load_server_config()
HOST = cfg["host"]
DOMAIN = cfg["domain"]

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(HOST, username="root", password=cfg["password"], timeout=15)

def run(cmd, timeout=120):
    print(f"$ {cmd[:120]}")
    _, stdout, stderr = c.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode()
    err = stderr.read().decode()
    rc = stdout.channel.recv_exit_status()
    if out.strip():
        for l in out.strip().split("\n")[:15]: print(f"  {l}")
    if err.strip() and rc != 0:
        for l in err.strip().split("\n")[:5]: print(f"  [err] {l}")
    return out, rc

# 1. Install nginx + certbot
print("=== [1/4] Installing nginx + certbot ===")
run("apt-get update -qq && apt-get install -y -qq nginx certbot python3-certbot-nginx", timeout=180)

# 2. Configure nginx for domain
print(f"\n=== [2/4] Configuring nginx for {DOMAIN} ===")
nginx_conf = f"""
server {{
    listen 80;
    server_name {DOMAIN};

    location / {{
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 120s;
        client_max_body_size 256k;
    }}
}}
"""
run(f"cat > /etc/nginx/sites-available/skillhub << 'NGINX_EOF'\n{nginx_conf}\nNGINX_EOF")
run("ln -sf /etc/nginx/sites-available/skillhub /etc/nginx/sites-enabled/skillhub")
run("rm -f /etc/nginx/sites-enabled/default")
run("fuser -k 80/tcp 2>/dev/null; sleep 1; systemctl restart nginx")

# 3. Get SSL cert
print("\n=== [3/4] Getting SSL certificate ===")
run(f"certbot --nginx -d {DOMAIN} --non-interactive --agree-tos --email admin@{DOMAIN.split('.', 1)[1]} --redirect 2>&1 || echo 'SSL attempted'", timeout=120)

# 4. Verify
print("\n=== [4/4] Verification ===")
run(f"curl -s https://{DOMAIN}/health")

c.close()
print(f"\nDone! https://{DOMAIN}")
