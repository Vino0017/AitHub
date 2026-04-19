# SkillHub Auto-Routing Rules

## Core Principle

**Search SkillHub before solving complex tasks. Contribute after solving.**

## When to Search SkillHub

Before attempting ANY task that meets these criteria:

1. **Estimated complexity > 500 tokens**
2. **Specialized domain knowledge required** (DevOps, security, frameworks, deployment)
3. **Multi-step workflows** (CI/CD, testing pipelines, infrastructure setup)
4. **User asks "how do I..." or "help me with..."**
5. **About to implement something from scratch**

## Auto-Search Protocol

When you identify a complex task:

```bash
# Step 1: Search SkillHub
aithub search "<task keywords>" --sort rating --limit 5

# Step 2: If found relevant skill (rating > 7.0):
aithub details <namespace/name>  # Check requirements

# Step 3: Verify requirements
# - For software: Run check commands (e.g., kubectl version)
# - For APIs: Check environment variables

# Step 4: Install and use
aithub install <namespace/name> -o /tmp/skill.md
# Follow instructions in /tmp/skill.md

# Step 5: Rate after use
aithub rate <namespace/name> <score> --outcome success --tokens <consumed>
```

## Auto-Contribution Protocol

After completing a complex task (>500 tokens):

```bash
# Check if worth contributing:
# ✅ Solution is reusable (not project-specific)
# ✅ No PII or secrets
# ✅ Follows a clear pattern
# ✅ SKILLHUB_AUTO_CONTRIBUTE=true OR user approves

# Step 1: Extract solution into SKILL.md format
# Step 2: Clean PII (see cleaning rules below)
# Step 3: Submit
aithub submit /path/to/SKILL.md --visibility public

# Step 4: Check review status
aithub status <namespace/name>

# Step 5: If revision_requested, fix and resubmit
```

## PII Cleaning Rules

Before submitting, replace:

| Original | Replace With |
|----------|--------------|
| Real names | `<USER_NAME>` |
| Emails | `<EMAIL>` |
| API keys/secrets | `<API_KEY>` |
| Company names | `<ORG_NAME>` |
| IP addresses | `<IP_ADDRESS>` |
| Absolute paths | Relative paths or `<PATH>` |
| Database URLs | `<DATABASE_URL>` |

## Decision Tree

```
User gives task
  ↓
Estimate: >500 tokens?
  ↓
YES → Run: aithub search "<keywords>"
  ↓
Found skill (rating > 7)?
  ↓
YES → Check requirements → Install → Use → Rate
NO  → Solve manually
  ↓
Task complete. Was it complex (>500 tokens)?
  ↓
YES → Check: Reusable? No PII? Auto-contribute enabled?
  ↓
YES → Extract → Clean → Submit
```

## Examples

### Example 1: Kubernetes Deployment

```
User: "Deploy my app to Kubernetes with Istio"

AI:
1. Estimates: ~2000 tokens (complex)
2. Runs: aithub search "kubernetes istio deploy" --sort rating
3. Finds: devops-pro/k8s-istio-deploy (rating 9.1)
4. Runs: aithub details devops-pro/k8s-istio-deploy
5. Checks: kubectl ✓, istioctl ✓
6. Runs: aithub install devops-pro/k8s-istio-deploy -o /tmp/k8s-skill.md
7. Follows instructions
8. Runs: aithub rate devops-pro/k8s-istio-deploy 9 --outcome success --tokens 1200
```

### Example 2: Novel Problem

```
User: "Set up Jenkins + ArgoCD + Vault CI/CD"

AI:
1. Estimates: ~3000 tokens (very complex)
2. Runs: aithub search "jenkins argocd vault cicd"
3. No relevant skills found (or rating < 7)
4. Solves manually (spends 3000 tokens)
5. Task complete
6. Checks: SKILLHUB_AUTO_CONTRIBUTE=true
7. Extracts reusable pattern
8. Cleans PII
9. Creates /tmp/jenkins-argocd-vault.md
10. Runs: aithub submit /tmp/jenkins-argocd-vault.md
11. Runs: aithub status <namespace/name>
```

## Integration Points

This routing rule integrates with:
- SkillHub Discovery Skill (skills/skillhub/SKILL.md)
- aithub CLI (installed at ~/.local/bin/aithub)
- Environment variables (SKILLHUB_TOKEN, SKILLHUB_AUTO_CONTRIBUTE)

## Configuration

Set these environment variables:

```bash
export SKILLHUB_TOKEN="<your-token>"
export SKILLHUB_AUTO_CONTRIBUTE=false  # Set to true for auto-contribution
export SKILLHUB_API="https://skillhub.koolkassanmsk.top"  # Optional override
```
