# 预制技能状态

## ✅ 已下载的技能集合

### gstack by Garry Tan
- **状态**: ✅ 已下载
- **位置**: `/opt/skillhub/skills/preloaded/gstack`
- **技能数量**: 45 个
- **GitHub**: https://github.com/garrytan/gstack
- **Stars**: 66K+

**包含的技能**:
- office-hours - YC Office Hours 模式
- plan-ceo-review - CEO/创始人模式计划审查
- plan-eng-review - 工程经理模式计划审查  
- plan-design-review - 设计师视角计划审查
- design-consultation - 完整设计系统咨询
- review - 代码审查
- ship - 完整发布工作流
- qa - 系统化 QA 测试
- investigate - 系统化调试
- 以及其他 36 个专业技能...

## 📋 技能使用方式

### 方式 1: 直接使用 (推荐)

gstack 技能已经下载到服务器，可以直接在 Claude Code 中使用：

```bash
# 在 Claude Code 中
/office-hours
/ship
/qa
/investigate
```

### 方式 2: 通过 SkillHub API

要通过 SkillHub API 使用这些技能，需要：

1. **注册用户并创建 namespace**:
   ```bash
   # 使用 GitHub OAuth 注册
   bash <(curl -fsSL https://aithub.space/install) --register --github
   ```

2. **获取 token**:
   ```bash
   # 注册后会自动获得 token
   export SKILLHUB_TOKEN="your-token-here"
   ```

3. **导入技能**:
   ```bash
   # 使用导入脚本
   TOKEN=$SKILLHUB_TOKEN bash /opt/skillhub/scripts/import_gstack_skills.sh
   ```

## 🔄 其他技能集合

### Everything Claude Code (计划中)
- **GitHub**: https://github.com/cline/everything-claude-code
- **Stars**: 100K+
- **状态**: ⏳ 待下载 (仓库可能不存在或私有)

### Agency Agents (计划中)
- **GitHub**: https://github.com/msitarzewski/agency-agents
- **Stars**: 2K+
- **状态**: ⏳ 待下载 (仓库可能不存在或私有)

## 📊 当前统计

```bash
# 查看 SkillHub 统计
curl https://aithub.space/v1/stats
```

当前数据库中的技能数量: 0 (技能文件已下载但未导入数据库)

## 🚀 快速开始

### 对于 Claude Code 用户

1. **克隆 gstack 到本地**:
   ```bash
   git clone https://github.com/garrytan/gstack.git ~/.claude/skills/gstack
   ```

2. **使用技能**:
   ```bash
   /office-hours
   /ship
   /qa
   ```

### 对于 SkillHub API 用户

1. **注册账号**:
   ```bash
   curl https://aithub.space/install | bash --register --github
   ```

2. **搜索技能**:
   ```bash
   curl "https://aithub.space/v1/skills?q=office+hours" \
     -H "Authorization: Bearer $SKILLHUB_TOKEN"
   ```

3. **使用技能**:
   ```bash
   curl "https://aithub.space/v1/skills/gstack/office-hours/content" \
     -H "Authorization: Bearer $SKILLHUB_TOKEN"
   ```

## 📝 注意事项

1. **技能文件已下载**: gstack 的 45 个技能已经下载到服务器
2. **数据库导入**: 需要注册用户才能将技能导入到 SkillHub 数据库
3. **直接使用**: 可以直接从文件系统使用这些技能，无需导入数据库
4. **权限要求**: 导入技能需要有效的用户 token 和 namespace

## 🔧 管理命令

```bash
# 查看已下载的技能
ls -la /opt/skillhub/skills/preloaded/gstack/

# 统计技能数量
find /opt/skillhub/skills/preloaded/gstack -name "SKILL.md" | wc -l

# 查看特定技能
cat /opt/skillhub/skills/preloaded/gstack/office-hours/SKILL.md

# 导入所有技能 (需要有效 token)
TOKEN=your-token bash /opt/skillhub/scripts/import_gstack_skills.sh
```

---

**最后更新**: 2026-04-20 14:10 UTC  
**状态**: ✅ gstack 技能已下载，可直接使用
