#!/usr/bin/env python3
"""
SkillHub Docker部署脚本 - 使用Docker Compose
"""

import paramiko
import os
import sys
import time

HOST = "192.227.235.131"
USER = "root"
PASSWORD = "Vino"
DOMAIN = "skillhub.koolkassanmsk.top"
REMOTE_DIR = "/opt/skillhub"
LLM_API_KEY = "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"

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
    print(f"SkillHub Docker Deploy to {HOST}")
    print("=" * 60)

    # 连接服务器
    print("\n[1/6] Connecting to server...")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    try:
        client.connect(HOST, username=USER, password=PASSWORD, timeout=15)
        print("    OK: SSH connected")
    except Exception as e:
        print(f"    ERROR: SSH failed: {e}")
        sys.exit(1)

    sftp = client.open_sftp()

    # 检查当前状态
    print("\n[2/6] Checking current status...")
    run_command(client, f"cd {REMOTE_DIR} && docker compose ps")
    run_command(client, f"ls -la {REMOTE_DIR}/ | head -10")

    # 上传所有源代码
    print("\n[3/6] Uploading source code...")
    project_dir = os.getcwd()
    print(f"    Local dir: {project_dir}")

    # 上传所有Go文件和配置
    files_to_sync = []
    for root, dirs, files in os.walk(project_dir):
        # 跳过不需要的目录
        dirs[:] = [d for d in dirs if d not in ['.git', 'node_modules', '.next', 'dist', '__pycache__']]

        for file in files:
            if file.endswith(('.go', '.sql', '.md', '.yml', '.yaml', '.mod', '.sum', '.sh', '.example')):
                local_path = os.path.join(root, file)
                rel_path = os.path.relpath(local_path, project_dir)
                remote_path = f"{REMOTE_DIR}/{rel_path.replace(os.sep, '/')}"

                # 确保远程目录存在
                remote_dir = os.path.dirname(remote_path)
                try:
                    sftp.stat(remote_dir)
                except:
                    run_command(client, f"mkdir -p {remote_dir}")

                # 上传文件
                try:
                    sftp.put(local_path, remote_path)
                    files_to_sync.append(rel_path)
                except Exception as e:
                    print(f"    WARN: {rel_path}: {e}")

    print(f"    Uploaded {len(files_to_sync)} files")

    # 更新环境变量
    print("\n[4/6] Updating environment...")

    # 生成新的ADMIN_TOKEN
    import secrets
    admin_token = secrets.token_hex(32)
    print(f"    New ADMIN_TOKEN: {admin_token}")

    # 保存到本地
    with open('.admin_token.txt', 'w') as f:
        f.write(admin_token)
    print("    Saved to .admin_token.txt")

    # 更新.env
    env_content = f"""DATABASE_URL=postgresql://skillhub:skillhub_password@localhost:5432/skillhub?sslmode=disable
PORT=8080
AUTO_MIGRATE=true
SEED_DATA=false
ADMIN_TOKEN={admin_token}
AI_REVIEW_ENABLED=true
LLM_PROVIDER=openrouter
LLM_API_KEY={LLM_API_KEY}
LLM_MODEL=google/gemma-4-31b-it
LLM_BASE_URL=https://openrouter.ai/api/v1
"""

    run_command(client, f"cat > {REMOTE_DIR}/.env << 'EOF'\n{env_content}\nEOF")

    # 重新构建和启动
    print("\n[5/6] Rebuilding and restarting...")
    run_command(client, f"cd {REMOTE_DIR} && docker compose down", timeout=60)
    time.sleep(3)
    run_command(client, f"cd {REMOTE_DIR} && docker compose up -d --build", timeout=600)

    # 等待服务启动
    print("\n[6/6] Waiting for service...")
    for i in range(30):
        time.sleep(3)
        out, _, _ = run_command(client, "curl -s http://localhost:8080/health 2>/dev/null || echo 'waiting'")
        if '"ok":true' in out:
            print("\n    === API IS LIVE! ===")
            break
    else:
        print("\n    Checking logs...")
        run_command(client, f"cd {REMOTE_DIR} && docker compose logs api --tail 50")

    # 测试
    print("\n=== Testing ===")
    run_command(client, "curl -s http://localhost:8080/health")
    run_command(client, "curl -s http://localhost:8080/v1/bootstrap/discovery | head -20")
    run_command(client, "curl -s http://localhost:8080/v1/bootstrap/check")
    run_command(client, "curl -s -X POST http://localhost:8080/v1/tokens")

    # 清理
    sftp.close()
    client.close()

    print("\n" + "=" * 60)
    print("Deploy complete!")
    print(f"URL: https://{DOMAIN}")
    print(f"Health: https://{DOMAIN}/health")
    print(f"ADMIN_TOKEN: {admin_token}")
    print("=" * 60)

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nDeploy interrupted")
        sys.exit(1)
    except Exception as e:
        print(f"\n\nDeploy failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
