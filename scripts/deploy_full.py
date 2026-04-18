import paramiko, os, time, sys

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
            if "密码" in line and ("：" in line or ":" in line):
                config["password"] = line.split("：" if "：" in line else ":")[1].strip()
            if "域名" in line and ("：" in line or ":" in line):
                config["domain"] = line.split("：" if "：" in line else ":")[1].strip()
    return config

cfg = load_server_config()
HOST = cfg["host"]
USER = "root"
PASS = cfg["password"]
PROJECT_DIR = os.path.join(os.path.dirname(__file__), "..")
REMOTE_DIR = "/opt/skillhub"

FILES = [
    "go.mod", "go.sum", "Dockerfile", "docker-compose.yml", ".env.example",
    "cmd/api/main.go",
    "internal/crypto/crypto.go",
    "internal/db/db.go",
    "internal/email/sender.go",
    "internal/handler/access.go", "internal/handler/admin.go", "internal/handler/auth.go", "internal/handler/fork.go",
    "internal/handler/namespace.go", "internal/handler/rating.go", "internal/handler/revision.go",
    "internal/handler/skill_detail.go", "internal/handler/skill_search.go",
    "internal/handler/skill_submit.go", "internal/handler/skill_yank.go",
    "internal/handler/token.go", "internal/handler/web.go",
    "internal/helpers/http.go", "internal/llm/llm.go",
    "internal/middleware/auth.go", "internal/models/models.go",
    "internal/review/regex_scanner.go", "internal/review/reviewer.go", "internal/review/worker.go",
    "internal/skillformat/validate.go", "internal/skillformat/semver.go",
    "migrations/001_create_namespaces.sql", "migrations/002_create_org_members.sql",
    "migrations/003_create_tokens.sql", "migrations/004_create_skills.sql",
    "migrations/005_create_revisions.sql", "migrations/006_create_ratings.sql",
    "migrations/007_create_auth_tables.sql",
    "scripts/seed.sql", "scripts/install.sh", "scripts/deploy.sh",
    "skills/skillhub/SKILL.md",
]

def run(client, cmd, timeout=120):
    print(f"  $ {cmd[:120]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode()
    err = stderr.read().decode()
    rc = stdout.channel.recv_exit_status()
    if out.strip():
        for l in out.strip().split("\n")[:15]: print(f"    {l}")
    if err.strip() and rc != 0:
        for l in err.strip().split("\n")[:10]: print(f"    [err] {l}")
    return out, rc

def main():
    print(f"=== Deploy to {HOST} ===")
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(HOST, username=USER, password=PASS, timeout=15)
    sftp = c.open_sftp()

    # 1. Create dirs
    print("\n[1/6] Creating directories...")
    dirs = set()
    for f in FILES:
        p = os.path.dirname(f)
        if p:
            full = f"{REMOTE_DIR}/{p}"
            parts = full.split("/")
            for i in range(2, len(parts)+1): dirs.add("/".join(parts[:i]))
    for d in sorted(dirs): run(c, f"mkdir -p {d}")

    # 2. Upload
    print(f"\n[2/6] Uploading {len(FILES)} files...")
    for f in FILES:
        local = os.path.join(PROJECT_DIR, f.replace("/", os.sep))
        if os.path.exists(local):
            sftp.put(local, f"{REMOTE_DIR}/{f}")
            print(f"    + {f}")

    # 3. Clean stale files
    print("\n[3/6] Cleaning stale files...")
    run(c, f"rm -f {REMOTE_DIR}/internal/crypto/hash.go {REMOTE_DIR}/migrations/001_initial.sql {REMOTE_DIR}/migrations/002_unique_skill_name_framework.sql")

    # 4. Setup .env
    print("\n[4/6] Setting up .env...")
    run(c, f"""
if [ ! -f {REMOTE_DIR}/.env ]; then
    cp {REMOTE_DIR}/.env.example {REMOTE_DIR}/.env
    ADMIN_TOKEN=$(head -c 32 /dev/urandom | xxd -p | tr -d '\\n')
    sed -i "s/ADMIN_TOKEN=change-me-in-production/ADMIN_TOKEN=$ADMIN_TOKEN/" {REMOTE_DIR}/.env
    echo 'Created .env'
else
    echo '.env exists'
fi
""")

    # 5. Reset DB + rebuild (only postgres + api, web is deployed separately)
    print("\n[5/6] Resetting DB and rebuilding...")
    run(c, f"cd {REMOTE_DIR} && docker compose down -v 2>/dev/null; sleep 2; docker compose up -d --build postgres api", timeout=600)

    # 6. Wait for health
    print("\n[6/6] Waiting for API...")
    for i in range(30):
        time.sleep(3)
        out, _ = run(c, "curl -s http://localhost:8080/health 2>/dev/null || echo 'waiting'")
        if '"ok":true' in out:
            print("\n    === API IS LIVE! ===")
            break
    else:
        print("\n    Checking logs...")
        run(c, f"cd {REMOTE_DIR} && docker compose logs api --tail 30")

    # Test
    print("\n=== Testing endpoints ===")
    run(c, "curl -s http://localhost:8080/health")
    run(c, "curl -s -X POST http://localhost:8080/v1/tokens")
    run(c, "curl -s 'http://localhost:8080/v1/skills' -H 'Authorization: Bearer test'")

    sftp.close()
    c.close()
    print(f"\nAPI: http://{HOST}:8080")

if __name__ == "__main__":
    main()
