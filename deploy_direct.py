#!/usr/bin/env python3
"""
SkillHub 快速部署脚本 - 直接部署Go二进制
使用密码认证SSH连接服务器
"""

import paramiko
import os
import sys
import time

# 服务器配置
HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
DOMAIN = "skillhub.koolkassanmsk.top"
REMOTE_DIR = "/opt/skillhub"
LLM_API_KEY = "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"

# 需要上传的文件
FILES_TO_UPLOAD = [
    "cmd/api/main.go",
    "go.mod",
    "go.sum",
    "internal/handler/bootstrap.go",
    "internal/handler/admin.go",
    "internal/handler/fork.go",
    "internal/handler/skill_detail.go",
    "internal/privacy/cleaner.go",
    "internal/usage/tracker.go",
    "internal/validation/environment.go",
    "internal/credibility/analyzer.go",
    "migrations/008_add_version_features.sql",
    "migrations/009_add_rating_credibility.sql",
    "migrations/010_add_usage_stats.sql",
]

def run_command(client, cmd, timeout=120):
    """执行SSH命令"""
    print(f"  $ {cmd[:100]}")
    stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore')
    err = stderr.read().decode('utf-8', errors='ignore')
    rc = stdout.channel.recv_exit_status()

    if out.strip():
        for line in out.strip().split('\n')[:20]:
            print(f"    {line}")
    if err.strip() and rc != 0:
        for line in err.strip().split('\n')[:10]:
            print(f"    [ERROR] {line}")

    return out, err, rc

def main():
    print(f"🚀 SkillHub 部署到 {HOST}")
    print("=" * 60)

    # 连接服务器
    print("\n[1/8] 连接服务器...")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        client.connect(HOST, username=USER, password=PASSWORD, timeout=15)
        print("    ✅ SSH连接成功")
    except Exception as e:
        print(f"    ❌ SSH连接失败: {e}")
        sys.exit(1)

    sftp = client.open_sftp()

    # 检查当前状态
    print("\n[2/8] 检查当前状态...")
    out, _, _ = run_command(client, "systemctl status skillhub --no-pager | head -5")
    out2, _, _ = run_command(client, "ls -la /opt/skillhub/ | head -10")

    # 创建备份
    print("\n[3/8] 备份当前版本...")
    run_command(client, f"cp {REMOTE_DIR}/skillhub {REMOTE_DIR}/skillhub.backup.$(date +%Y%m%d_%H%M%S) 2>/dev/null || echo 'No backup needed'")

    # 上传所有源代码文件
    print(f"\n[4/8] 上传源代码文件...")
    project_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

    # 确保目录存在
    dirs_to_create = set()
    for file_path in FILES_TO_UPLOAD:
        dir_path = os.path.dirname(file_path)
        if dir_path:
            dirs_to_create.add(dir_path)

    for dir_path in sorted(dirs_to_create):
        run_command(client, f"mkdir -p {REMOTE_DIR}/{dir_path}")

    # 上传文件
    uploaded = 0
    for file_path in FILES_TO_UPLOAD:
        local_path = os.path.join(project_dir, file_path.replace('/', os.sep))
        remote_path = f"{REMOTE_DIR}/{file_path}"

        if os.path.exists(local_path):
            try:
                sftp.put(local_path, remote_path)
                print(f"    ✅ {file_path}")
                uploaded += 1
            except Exception as e:
                print(f"    ⚠️  {file_path}: {e}")
        else:
            print(f"    ⚠️  本地文件不存在: {file_path}")

    print(f"    上传完成: {uploaded}/{len(FILES_TO_UPLOAD)} 个文件")

    # 上传所有internal目录（确保完整）
    print("\n[5/8] 同步internal目录...")
    internal_dir = os.path.join(project_dir, "internal")
    if os.path.exists(internal_dir):
        for root, dirs, files in os.walk(internal_dir):
            for file in files:
                if file.endswith('.go'):
                    local_file = os.path.join(root, file)
                    rel_path = os.path.relpath(local_file, project_dir)
                    remote_file = f"{REMOTE_DIR}/{rel_path.replace(os.sep, '/')}"

                    # 确保远程目录存在
                    remote_dir = os.path.dirname(remote_file)
                    run_command(client, f"mkdir -p {remote_dir}")

                    try:
                        sftp.put(local_file, remote_file)
                    except:
                        pass

    # 上传迁移文件
    print("\n[6/8] 上传数据库迁移...")
    migrations_dir = os.path.join(project_dir, "migrations")
    if os.path.exists(migrations_dir):
        run_command(client, f"mkdir -p {REMOTE_DIR}/migrations")
        for file in os.listdir(migrations_dir):
            if file.endswith('.sql'):
                local_file = os.path.join(migrations_dir, file)
                remote_file = f"{REMOTE_DIR}/migrations/{file}"
                try:
                    sftp.put(local_file, remote_file)
                    print(f"    ✅ {file}")
                except Exception as e:
                    print(f"    ⚠️  {file}: {e}")

    # 在服务器上构建
    print("\n[7/8] 在服务器上构建...")
    out, err, rc = run_command(client, f"""
cd {REMOTE_DIR}
go build -o skillhub cmd/api/main.go
chmod +x skillhub
ls -lh skillhub
""", timeout=300)

    if rc != 0:
        print("    ❌ 构建失败！")
        print(f"    错误: {err}")
        sftp.close()
        client.close()
        sys.exit(1)

    print("    ✅ 构建成功")

    # 更新环境变量
    print("\n[8/8] 更新环境变量并重启服务...")

    # 生成新的ADMIN_TOKEN
    import secrets
    admin_token = secrets.token_hex(32)
    print(f"    🔑 新的ADMIN_TOKEN: {admin_token}")

    # 保存到本地
    with open('.admin_token.txt', 'w') as f:
        f.write(admin_token)
    print("    ✅ ADMIN_TOKEN已保存到 .admin_token.txt")

    # 更新服务器上的.env
    env_content = f"""# Database
DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable

# Server
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false

# Admin
ADMIN_TOKEN={admin_token}

# AI Review
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY={LLM_API_KEY}
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1
"""

    # 写入.env文件
    run_command(client, f"cat > {REMOTE_DIR}/.env << 'EOF'\n{env_content}\nEOF")

    # 重启服务
    print("\n    重启服务...")
    run_command(client, "systemctl restart skillhub")

    # 等待服务启动
    print("\n    等待服务启动...")
    time.sleep(5)

    # 检查服务状态
    print("\n    检查服务状态...")
    run_command(client, "systemctl status skillhub --no-pager | head -15")

    # 测试健康检查
    print("\n=== 测试部署结果 ===")
    time.sleep(2)

    print("\n1. 健康检查:")
    run_command(client, "curl -s http://localhost:8080/health")

    print("\n2. Bootstrap Discovery:")
    run_command(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")

    print("\n3. Bootstrap Check:")
    run_command(client, "curl -s http://localhost:8080/v1/bootstrap/check")

    print("\n4. 创建Token:")
    out, _, _ = run_command(client, "curl -s -X POST http://localhost:8080/v1/tokens -H 'Content-Type: application/json' -d '{}'")

    # 提取token进行进一步测试
    if '"token":"' in out:
        import json
        try:
            token_data = json.loads(out)
            token = token_data.get('token', '')
            if token:
                print(f"\n5. 使用Token测试搜索:")
                run_command(client, f"curl -s -H 'Authorization: Bearer {token}' 'http://localhost:8080/v1/skills?limit=3'")
        except:
            pass

    # 查看最近日志
    print("\n=== 最近日志 ===")
    run_command(client, "journalctl -u skillhub -n 30 --no-pager")

    # 清理
    sftp.close()
    client.close()

    print("\n" + "=" * 60)
    print("✨ 部署完成！")
    print(f"🌐 服务地址: https://{DOMAIN}")
    print(f"🏥 健康检查: https://{DOMAIN}/health")
    print(f"🔑 ADMIN_TOKEN: {admin_token} (已保存到 .admin_token.txt)")
    print("=" * 60)

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n⚠️  部署被用户中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n\n❌ 部署失败: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
