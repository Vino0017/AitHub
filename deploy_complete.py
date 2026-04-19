#!/usr/bin/env python3
"""完整部署脚本 - 包含所有新功能文件"""

import paramiko
import os
import time
import sys

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"
PROJECT_DIR = os.path.dirname(os.path.abspath(__file__))

# 完整文件列表 - 包含所有新功能
FILES = [
    "go.mod", "go.sum", "Dockerfile", "docker-compose.yml", ".env.example", ".dockerignore",
    "cmd/api/main.go",
    # 所有handler文件
    "internal/handler/admin.go", "internal/handler/auth.go", "internal/handler/bootstrap.go",
    "internal/handler/fork.go", "internal/handler/namespace.go", "internal/handler/rating.go",
    "internal/handler/revision.go", "internal/handler/skill_detail.go", "internal/handler/skill_search.go",
    "internal/handler/skill_submit.go", "internal/handler/skill_yank.go", "internal/handler/token.go",
    "internal/handler/web.go",
    # 新功能模块
    "internal/privacy/cleaner.go",
    "internal/usage/tracker.go",
    "internal/validation/environment.go",
    "internal/credibility/analyzer.go",
    # 其他internal文件
    "internal/crypto/crypto.go", "internal/db/db.go", "internal/email/sender.go",
    "internal/helpers/http.go", "internal/llm/llm.go", "internal/middleware/auth.go",
    "internal/models/models.go",
    "internal/review/regex_scanner.go", "internal/review/reviewer.go", "internal/review/worker.go",
    "internal/skillformat/validate.go", "internal/skillformat/semver.go",
    # 所有迁移文件
    "migrations/001_create_namespaces.sql", "migrations/002_create_org_members.sql",
    "migrations/003_create_tokens.sql", "migrations/004_create_skills.sql",
    "migrations/005_create_revisions.sql", "migrations/006_create_ratings.sql",
    "migrations/007_create_auth_tables.sql", "migrations/008_add_version_features.sql",
    "migrations/009_add_rating_credibility.sql", "migrations/010_add_usage_stats.sql",
    # 脚本
    "scripts/seed.sql", "scripts/install.sh",
    "skills/skillhub/SKILL.md",
]

def run(client, cmd, timeout=120):
    print(f"  $ {cmd[:100]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')
    rc = stdout.channel.recv_exit_status()
    if out.strip():
        for l in out.strip().split("\n")[:15]:
            print(f"    {l}")
    if err.strip() and rc != 0:
        for l in err.strip().split("\n")[:10]:
            print(f"    [ERR] {l}")
    return out, rc

def main():
    print(f"=== Complete Deploy to {HOST} ===\n")

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASSWORD, timeout=15)
    sftp = client.open_sftp()

    # 1. 创建目录
    print("[1/7] Creating directories...")
    dirs = set()
    for f in FILES:
        p = os.path.dirname(f)
        if p:
            full = f"{REMOTE_DIR}/{p}"
            parts = full.split("/")
            for i in range(2, len(parts)+1):
                dirs.add("/".join(parts[:i]))
    for d in sorted(dirs):
        run(client, f"mkdir -p {d}")

    # 2. 上传文件
    print(f"\n[2/7] Uploading {len(FILES)} files...")
    uploaded = 0
    for f in FILES:
        local = os.path.join(PROJECT_DIR, f.replace("/", os.sep))
        if os.path.exists(local):
            sftp.put(local, f"{REMOTE_DIR}/{f}")
            print(f"    + {f}")
            uploaded += 1
        else:
            print(f"    ! Missing: {f}")
    print(f"    Uploaded: {uploaded}/{len(FILES)}")

    # 3. 设置环境变量
    print("\n[3/7] Setting up .env...")
    import secrets
    admin_token = secrets.token_hex(32)
    print(f"    New ADMIN_TOKEN: {admin_token}")

    env_content = f"""DATABASE_URL=postgresql://skillhub:skillhub_password@postgres:5432/skillhub?sslmode=disable
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false
ADMIN_TOKEN={admin_token}
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY=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
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1
"""
    run(client, f"cat > {REMOTE_DIR}/.env << 'EOF'\n{env_content}\nEOF")

    # 保存到本地
    with open('.admin_token_new.txt', 'w') as f:
        f.write(admin_token)

    # 4. 停止旧容器
    print("\n[4/7] Stopping old containers...")
    run(client, f"cd {REMOTE_DIR} && docker compose down")
    time.sleep(3)

    # 5. 构建新镜像
    print("\n[5/7] Building new image...")
    out, rc = run(client, f"cd {REMOTE_DIR} && docker compose build --no-cache api", timeout=600)

    if rc != 0:
        print("\n    Build failed! Checking details...")
        run(client, f"cd {REMOTE_DIR} && ls -la")
        run(client, f"cd {REMOTE_DIR} && cat Dockerfile")
        sftp.close()
        client.close()
        sys.exit(1)

    # 6. 启动服务
    print("\n[6/7] Starting services...")
    run(client, f"cd {REMOTE_DIR} && docker compose up -d", timeout=300)

    # 7. 等待并测试
    print("\n[7/7] Waiting for API...")
    for i in range(30):
        time.sleep(3)
        out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
        if '"ok":true' in out:
            print("\n    === API IS LIVE! ===")
            break
    else:
        print("\n    Timeout! Checking logs...")
        run(client, f"cd {REMOTE_DIR} && docker compose logs api --tail 50")

    # 测试新功能
    print("\n=== Testing New Features ===")
    run(client, "curl -s http://localhost:8080/health")
    run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
    run(client, "curl -s http://localhost:8080/v1/bootstrap/check")
    run(client, "curl -s -X POST http://localhost:8080/v1/tokens")

    sftp.close()
    client.close()

    print(f"\n=== Deploy Complete! ===")
    print(f"URL: https://skillhub.koolkassanmsk.top")
    print(f"ADMIN_TOKEN: {admin_token}")

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"\nDeploy failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
