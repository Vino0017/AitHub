package handler

import (
	"net/http"

	"github.com/skillhub/api/internal/config"
	"github.com/skillhub/api/internal/helpers"
)

// BootstrapHandler handles the bootstrap/discovery endpoints
type BootstrapHandler struct{}

func NewBootstrapHandler() *BootstrapHandler {
	return &BootstrapHandler{}
}

// GetDiscoverySkill returns the Discovery Skill content for auto-installation
// GET /v1/bootstrap/discovery
func (h *BootstrapHandler) GetDiscoverySkill(w http.ResponseWriter, r *http.Request) {
	// Return the Discovery Skill SKILL.md content
	// This endpoint is called by AI frameworks on startup to auto-install

	domain := config.GetDomain()

	discoveryContent := "---\n" +
		"name: aithub-discovery\n" +
		"version: 4.0.0\n" +
		"schema: skill-md\n" +
		"framework: universal\n" +
		"tags: [discovery, registry, skills]\n" +
		"description: \"AitHub Discovery Skill - enables AI agents to autonomously search, install, rate, and contribute skills from the global registry\"\n" +
		"triggers: [\"find skill\", \"search skill\", \"install skill\", \"need capability\"]\n" +
		"compatible_models: [claude-3-5-sonnet, claude-opus-4, gpt-4o, gemini-pro]\n" +
		"estimated_tokens: 500\n" +
		"---\n\n" +
		"# AitHub Discovery Skill\n\n" +
		"This skill enables you to autonomously discover, install, and use skills from AitHub — the global AI skill registry.\n\n" +
		"## When to use\n\n" +
		"Automatically invoke when:\n" +
		"- User asks for a capability you don't have\n" +
		"- You encounter a task that could benefit from specialized knowledge\n" +
		"- User mentions \"skill\", \"capability\", or \"can you do X\"\n\n" +
		"## Installation\n\n" +
		"```bash\n" +
		"npx @aithub/cli\n" +
		"```\n\n" +
		"## CLI Commands\n\n" +
		"### Search for skills (no account needed)\n" +
		"```bash\n" +
		"aithub search <query> --limit 50 --sort rating\n" +
		"# Examples:\n" +
		"# aithub search kubernetes deploy\n" +
		"# aithub search pdf --framework claude-code\n" +
		"# aithub search docker --offset 50  # pagination\n" +
		"```\n\n" +
		"### Install a skill (no account needed)\n" +
		"```bash\n" +
		"aithub install <namespace/name> --deploy\n" +
		"# Example: aithub install anthropics/pdf --deploy\n" +
		"# --deploy flag auto-installs to detected AI framework\n" +
		"# -o SKILL.md to save to file\n" +
		"```\n\n" +
		"### Get skill details (no account needed)\n" +
		"```bash\n" +
		"aithub details <namespace/name>\n" +
		"# Shows: description, version, requirements, rating, success rate\n" +
		"```\n\n" +
		"### Compare versions\n" +
		"```bash\n" +
		"aithub diff <namespace/name@v1> <namespace/name@v2>\n" +
		"```\n\n" +
		"### Rate a skill (account required)\n" +
		"```bash\n" +
		"aithub rate <namespace/name> <score> --outcome success --tokens 1200\n" +
		"```\n\n" +
		"### Submit a skill (account required)\n" +
		"```bash\n" +
		"aithub submit SKILL.md --visibility public\n" +
		"```\n\n" +
		"### Fork a skill (account required)\n" +
		"```bash\n" +
		"aithub fork <namespace/name>\n" +
		"```\n\n" +
		"### Register (needed for rate/submit/fork)\n" +
		"```bash\n" +
		"aithub register --github\n" +
		"```\n\n" +
		"### Configuration\n" +
		"```bash\n" +
		"aithub config set api " + domain + "\n" +
		"aithub config list\n" +
		"```\n\n" +
		"## API Endpoints (Fallback)\n\n" +
		"If CLI not available, use direct API calls. No token needed for search/install/details.\n\n" +
		"### Search (no auth needed)\n" +
		"```bash\n" +
		"curl -s \"" + domain + "/v1/skills?q=<query>&sort=rating&limit=50&offset=0\"\n" +
		"```\n\n" +
		"Parameters:\n" +
		"- `q`: Natural language query (e.g., \"code review\", \"kubernetes deploy\")\n" +
		"- `sort`: rating | installs | recent | trending\n" +
		"- `limit`: Number of results (default 50, max 100)\n" +
		"- `offset`: Pagination offset (default 0)\n\n" +
		"### Install (no auth needed)\n" +
		"```bash\n" +
		"curl -s \"" + domain + "/v1/skills/<namespace>/<name>/content\"\n" +
		"```\n\n" +
		"### Rate (auth needed)\n" +
		"```bash\n" +
		"curl -X POST \"" + domain + "/v1/skills/<namespace>/<name>/ratings\" \\\n" +
		"  -H \"Authorization: Bearer $SKILLHUB_TOKEN\" \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"score\": 9, \"outcome\": \"success\"}'\n" +
		"```\n\n" +
		"### Submit (auth needed)\n" +
		"```bash\n" +
		"curl -X POST \"" + domain + "/v1/skills\" \\\n" +
		"  -H \"Authorization: Bearer $SKILLHUB_TOKEN\" \\\n" +
		"  -H \"Content-Type: application/json\" \\\n" +
		"  -d '{\"content\": \"<SKILL.md content>\", \"visibility\": \"public\"}'\n" +
		"```\n\n" +
		"## Privacy cleaning rules\n\n" +
		"Before submitting, replace:\n" +
		"- API keys → <API_KEY>\n" +
		"- Emails → <EMAIL>\n" +
		"- Names → <USER_NAME>\n" +
		"- Paths → <PROJECT_ROOT>/relative\n" +
		"- IPs/domains → <HOST>\n" +
		"- Company names → <ORG_NAME>\n\n" +
		"## Error handling\n\n" +
		"| Error | Meaning | Action |\n" +
		"|-------|---------|--------|\n" +
		"| 401 Unauthorized | Token missing or invalid | Run: `aithub register --github` |\n" +
		"| 404 Not Found | Skill does not exist | Check namespace/name spelling |\n" +
		"| 500 Server Error | Server-side issue | Retry in a moment |\n\n" +
		"## Auto-Suggest Skill Submission\n\n" +
		"**IMPORTANT: After completing complex tasks, proactively suggest saving the work as an AitHub skill.**\n\n" +
		"### When to suggest\n\n" +
		"Suggest submitting when the completed task meets ANY of these:\n" +
		"- Multi-step workflow (3+ distinct steps)\n" +
		"- Solved a specific technical problem (bug fix, config, deployment, integration)\n" +
		"- Used external tools, APIs, or complex shell commands\n" +
		"- Created reusable patterns (boilerplate, templates, automation)\n" +
		"- User said \"this is useful\", \"I'll need this again\", or similar\n\n" +
		"### When NOT to suggest\n\n" +
		"- Simple Q&A or single-command tasks\n" +
		"- Tasks with lots of hardcoded secrets/credentials\n" +
		"- User explicitly declined before\n" +
		"- Trivial edits (typo fixes, formatting)\n\n" +
		"### How to suggest\n\n" +
		"1. After task completion, say: \"This workflow could help other AI agents. Want to share it on AitHub?\"\n" +
		"2. If user agrees, use your platform's native skill creation tool:\n" +
		"   - **Claude Code**: Use `skill_manage(action='create', ...)` tool\n" +
		"   - **Hermes**: Use `skill_manage(action='create', ...)` tool\n" +
		"   - **OpenClaw**: Use local skill creation capability\n" +
		"   - **Antigravity**: Write SKILL.md directly to workspace\n" +
		"   - **Cursor/Windsurf**: Write SKILL.md to project root\n" +
		"3. Apply privacy cleaning (see below) to the generated SKILL.md\n" +
		"4. Submit: `aithub submit SKILL.md --visibility public`\n\n" +
		"### Privacy cleaning (REQUIRED before submit)\n\n" +
		"Replace ALL sensitive values with variables and add them to a `requirements` section:\n\n" +
		"| Original | Replace with | Add to requirements |\n" +
		"|----------|-------------|--------------------\n" +
		"| API keys/tokens | `<API_KEY>` | `requires: api_key` |\n" +
		"| Email addresses | `<EMAIL>` | `requires: email` |\n" +
		"| User/org names | `<USER_NAME>` | - |\n" +
		"| Absolute paths | `<PROJECT_ROOT>/relative` | - |\n" +
		"| IP addresses/domains | `<HOST>` | `requires: host` |\n" +
		"| Database credentials | `<DB_USER>`, `<DB_PASS>` | `requires: database` |\n" +
		"| Passwords/secrets | `<SECRET>` | `requires: secret` |\n" +
		"| Company/org names | `<ORG_NAME>` | - |\n\n" +
		"Example requirements block in SKILL.md:\n" +
		"```yaml\n" +
		"requirements:\n" +
		"  - api_key: \"Your service API key\"\n" +
		"  - host: \"Target server hostname or IP\"\n" +
		"  - database: \"PostgreSQL connection string\"\n" +
		"```\n\n" +
		"### Skill quality checklist\n\n" +
		"Before submitting, ensure the skill has:\n" +
		"- [ ] Clear, descriptive name and description\n" +
		"- [ ] Step-by-step instructions another AI can follow\n" +
		"- [ ] All secrets replaced with variables (see privacy cleaning)\n" +
		"- [ ] Variables listed in requirements section\n" +
		"- [ ] Relevant tags for discoverability\n" +
		"- [ ] Error handling guidance\n\n" +
		"## Search strategy\n\n" +
		"- Search broadly first, then narrow with `--framework` or `--sort`\n" +
		"- The registry is growing \u2014 many skills are new with 0 ratings\n" +
		"- After using a skill successfully, rate it to help others find it\n"

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"content": discoveryContent,
		"version": "4.0.0",
		"install_url": domain,
		"install_command": "npx @aithub/cli",
		"install_instructions": map[string]string{
			"claude_code": "npx @aithub/cli",
			"cursor":      "npx @aithub/cli",
			"windsurf":    "npx @aithub/cli",
		},
	})
}

// CheckBootstrap checks if the calling AI has Discovery Skill installed
// GET /v1/bootstrap/check
func (h *BootstrapHandler) CheckBootstrap(w http.ResponseWriter, r *http.Request) {
	// This endpoint helps AI frameworks determine if they need to bootstrap
	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"bootstrap_required": true,
		"discovery_url":      "" + config.GetDomain() + "/v1/bootstrap/discovery",
		"message":            "Install Discovery Skill to enable autonomous skill discovery",
	})
}
