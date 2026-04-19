#!/usr/bin/env python3
"""
SkillHub 完整部署脚本
从 GitHub 仓库拉取代码并部署前端和后端
"""
import paramiko, os, time, sys

# 服务器配置
HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
REMOTE_DIR = "/opt/skillhub"
REPO_URL = "https://github.com/Vino0017/AitHub.git"

def run(client, cmd, timeout=120):
    """执行远程命令并打印输出"""
    print(f"$ {cmd[:150]}")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')

    # 打印输出（最后25行）
    for line in out.strip().split("\n")[-25:] if out.strip() else []:
        try: print(f"  {line}")
        except: pass

    # 打印错误（最后10行）
    for line in err.strip().split("\n")[-10:] if err.strip() and "Pulling" not in err else []:
        try: print(f"  [ERR] {line}")
        except: pass

    return out, err

# 连接服务器
print("=== SkillHub 部署 ===\n")
print(f"[连接] {HOST}...")
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(HOST, username=USER, password=PASSWORD, timeout=15)

print("\n[1] 从 GitHub 拉取最新代码...")
out, err = run(client, f"""
cd {REMOTE_DIR} && \
git fetch origin && \
git reset --hard origin/main && \
git clean -fd
""", timeout=60)

if "error" in err.lower() or "fatal" in err.lower():
    print(f"\n⚠️  Git 拉取失败，尝试重新克隆...")
    run(client, f"rm -rf {REMOTE_DIR} && mkdir -p {REMOTE_DIR}")
    out, err = run(client, f"git clone {REPO_URL} {REMOTE_DIR}", timeout=120)
    if "error" in err.lower() or "fatal" in err.lower():
        print(f"\n❌ 克隆失败!\n{err[:500]}")
        client.close()
        sys.exit(1)

print("\n[2] 构建后端...")
out, err = run(client, f"""
cd {REMOTE_DIR} && \
docker run --rm -v $(pwd):/app -w /app golang:1.25-alpine sh -c \
"go build -o skillhub ./cmd/api && chmod +x skillhub && ls -lh skillhub"
""", timeout=300)

if "skillhub" not in out:
    print(f"\n❌ 后端构建失败!\n{err[:500]}")
    client.close()
    sys.exit(1)
print("  ✅ 后端构建成功!")

print("\n[3] 构建前端...")
out, err = run(client, f"""
cd {REMOTE_DIR}/web && \
npm install && \
npm run build
""", timeout=600)

if "error" in err.lower() and "warn" not in err.lower():
    print(f"\n⚠️  前端构建可能有问题:\n{err[:500]}")
else:
    print("  ✅ 前端构建成功!")

print("\n[4] 构建 Docker 镜像...")
run(client, f"cd {REMOTE_DIR} && docker compose build", timeout=300)

print("\n[5] 启动服务...")
run(client, f"cd {REMOTE_DIR} && docker compose up -d")

print("\n[6] 等待服务启动...")
time.sleep(25)

print("\n[7] 健康检查...")
for i in range(15):
    out, _ = run(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'wait'")
    if '"ok":true' in out:
        print("\n  ✅ 后端健康检查通过!")
        break
    time.sleep(2)
else:
    print("\n  ⚠️  后端健康检查超时")

print("\n[8] 测试 Bootstrap 端点...")
out, _ = run(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
if "SkillHub Discovery Skill" in out or "skill" in out.lower():
    print("  ✅ Bootstrap 端点工作正常!")
else:
    print(f"  ⚠️  Bootstrap 端点响应异常:\n{out[:200]}")

print("\n[9] 测试前端...")
out, _ = run(client, "curl -s http://localhost:3000 | head -20")
if "SkillHub" in out or "<!DOCTYPE" in out:
    print("  ✅ 前端服务正常!")
else:
    print(f"  ⚠️  前端响应异常:\n{out[:200]}")

client.close()

print("\n" + "="*50)
print("✅ 部署完成!")
print("="*50)
print(f"后端 API: https://skillhub.koolkassanmsk.top")
print(f"前端网页: https://skillhub.koolkassanmsk.top (如果配置了)")
print("="*50)
