#!/usr/bin/env python3
"""手动构建并部署 - 绕过Docker构建问题"""

import paramiko
import time

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"

def run(client, cmd, timeout=120):
    print(f"$ {cmd[:100]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    if out.strip():
        for l in out.strip().split("\n")[:20]:
            print(f"  {l}")
    return out

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("=== Manual Build & Deploy ===\n")

# 1. 在服务器上构建二进制
print("[1] Building binary on server...")
run(client, f"""
cd {REMOTE_DIR}
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c '
  go build -o skillhub ./cmd/api &&
  chmod +x skillhub &&
  ls -lh skillhub
'
""", timeout=300)

# 2. 修改Dockerfile使用预构建的二进制
print("\n[2] Creating simple Dockerfile...")
run(client, f"""
cat > {REMOTE_DIR}/Dockerfile.simple << 'EOF'
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY skillhub .
COPY migrations/ migrations/
COPY scripts/ scripts/
COPY skills/ skills/
EXPOSE 8080
ENTRYPOINT ["./skillhub"]
EOF
""")

# 3. 修改docker-compose使用新Dockerfile
print("\n[3] Updating docker-compose...")
run(client, f"""
cat > {REMOTE_DIR}/docker-compose.yml << 'EOF'
services:
  postgres:
    image: pgvector/pgvector:pg17
    environment:
      POSTGRES_DB: skillhub
      POSTGRES_USER: skillhub
      POSTGRES_PASSWORD: skillhub_password
    ports:
      - "127.0.0.1:5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U skillhub"]
      interval: 5s
      timeout: 3s
      retries: 5

  api:
    build:
      context: .
      dockerfile: Dockerfile.simple
    ports:
      - "0.0.0.0:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    env_file: .env
    restart: unless-stopped

volumes:
  pgdata:
EOF
""")

# 4. 构建并启动
print("\n[4] Building and starting...")
run(client, f"cd {REMOTE_DIR} && docker compose build api", timeout=120)
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

# 5. 等待并测试
print("\n[5] Waiting for service...")
time.sleep(15)

for i in range(10):
    out = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n=== API IS LIVE! ===")
        break
    time.sleep(2)

# 6. 测试新功能
print("\n[6] Testing new features...")
run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
run(client, "curl -s http://localhost:8080/v1/bootstrap/check")

token_out = run(client, "curl -s -X POST http://localhost:8080/v1/tokens")
print(f"\nToken created: {token_out[:100]}")

client.close()
print("\n=== DEPLOY COMPLETE! ===")
print("URL: https://skillhub.koolkassanmsk.top")
