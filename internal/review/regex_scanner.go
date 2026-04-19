package review

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/skillhub/api/internal/security"
)

// ScanIssue represents a finding from the regex scanner.
type ScanIssue struct {
	Type   string `json:"type"`
	Line   int    `json:"line"`
	Detail string `json:"detail"`
}

// Patterns based on gitleaks rules + modern API key formats
var secretPatterns = []struct {
	Name    string
	Pattern *regexp.Regexp
}{
	{"AWS Access Key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"GitHub Token", regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`)},
	{"GitHub OAuth", regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`)},
	{"GitHub PAT", regexp.MustCompile(`github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}`)},
	{"OpenAI API Key", regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}T3BlbkFJ[a-zA-Z0-9]{20,}`)},
	{"OpenAI Project Key", regexp.MustCompile(`sk-proj-[a-zA-Z0-9]{40,}`)},
	{"OpenRouter Key", regexp.MustCompile(`sk-or-v1-[a-zA-Z0-9]{60,}`)},
	{"Anthropic Key", regexp.MustCompile(`sk-ant-[a-zA-Z0-9-]{40,}`)},
	{"Groq Key", regexp.MustCompile(`gsk_[a-zA-Z0-9]{40,}`)},
	{"Google AI Key", regexp.MustCompile(`AIza[a-zA-Z0-9_-]{35}`)},
	{"Stripe Key", regexp.MustCompile(`sk_(live|test)_[a-zA-Z0-9]{24,}`)},
	{"Generic Long SK Key", regexp.MustCompile(`sk_[a-zA-Z0-9]{40,}`)},
	{"Slack Token", regexp.MustCompile(`xox[baprs]-[0-9]{10,}-[a-zA-Z0-9-]+`)},
	{"Private Key", regexp.MustCompile(`-----BEGIN (RSA|EC|DSA|OPENSSH|PGP) PRIVATE KEY-----`)},
	{"Generic Secret", regexp.MustCompile(`(?i)(password|passwd|secret|api_key|apikey|access_token)\s*[:=]\s*['"][^'"]{8,}['"]`)},
	{"JWT Secret", regexp.MustCompile(`(?i)jwt[_-]?secret\s*[:=]\s*['"][^'"]{8,}['"]`)},
	{"API Key in URL", regexp.MustCompile(`(?i)(api[_-]?key|apikey|token)=[a-zA-Z0-9_-]{20,}`)},
	{"Bearer Token", regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-.]{20,}`)},
	{"Connection String", regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^\s'"]+:[^\s'"]+@`)},
}

// RegexScan scans content for known secret patterns.
// Returns issues found. If any are found, the review can short-circuit LLM.
func RegexScan(content string) []ScanIssue {
	var issues []ScanIssue
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Skip YAML frontmatter field definitions (these are templates, not real secrets)
		// BUT: only skip if they are simple field declarations without actual secret values
		trimmed := strings.TrimSpace(line)
		if isYAMLFieldDeclaration(trimmed) {
			continue
		}

		for _, p := range secretPatterns {
			if p.Pattern.MatchString(line) {
				match := p.Pattern.FindString(line)
				// Truncate the match for display
				display := match
				if len(display) > 20 {
					display = display[:20] + "..."
				}
				issues = append(issues, ScanIssue{
					Type:   "privacy",
					Line:   lineNum + 1,
					Detail: fmt.Sprintf("%s detected: %s", p.Name, display),
				})
			}
		}
	}

	return issues
}

// isYAMLFieldDeclaration 检查是否是 YAML 字段声明（而不是实际的秘密值）
func isYAMLFieldDeclaration(line string) bool {
	// 只有当行是简单的字段名声明时才跳过
	// 例如: "env_var: OPENAI_API_KEY" (字段名)
	// 但不跳过: "env_var: sk-proj-abc123" (实际秘密)

	if !strings.HasPrefix(line, "env_var:") && !strings.HasPrefix(line, "obtain_url:") {
		return false
	}

	// 提取冒号后的值
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return false
	}

	value := strings.TrimSpace(parts[1])

	// 先检查是否包含秘密模式
	for _, p := range secretPatterns {
		if p.Pattern.MatchString(value) {
			return false // 包含秘密，不跳过
		}
	}

	// 如果值看起来像环境变量名（全大写，下划线，但不是 AWS key 格式），则是字段声明
	// 例如: "OPENAI_API_KEY", "DATABASE_URL"
	// 但不包括: "AKIAIOSFODNN7EXAMPLE" (AWS key)
	if regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(value) {
		// 额外检查：不是 AWS key 格式（AKIA 开头）
		if !strings.HasPrefix(value, "AKIA") {
			return true
		}
	}

	// 如果值是 URL 但不包含秘密模式，则是字段声明
	// 例如: "https://platform.openai.com/api-keys"
	if strings.HasPrefix(value, "http") {
		return true // 已经在上面检查过秘密模式了
	}

	// 其他情况不跳过
	return false
}

// MaliciousPatterns checks for dangerous commands.
var maliciousPatterns = []struct {
	Name    string
	Pattern *regexp.Regexp
}{
	{"Destructive rm", regexp.MustCompile(`rm\s+-[rR]f\s+/`)},
	{"Reverse Shell", regexp.MustCompile(`(?i)(bash\s+-i|/dev/tcp/|nc\s+-e|ncat\s+-e|python\s+-c\s+.*socket)`)},
	{"Data Exfil curl", regexp.MustCompile(`(?i)curl\s+.*\$\(.*\)`)},
	{"Crypto Miner", regexp.MustCompile(`(?i)(xmrig|minerd|stratum\+tcp|coinhive)`)},
	{"Disk Wipe", regexp.MustCompile(`(?i)(dd\s+if=/dev/zero|mkfs\.|format\s+[A-Z]:)`)},
	{"Eval Injection", regexp.MustCompile(`(?i)eval\s*\(\s*\$`)},
}

// SecurityScan checks for malicious commands.
func SecurityScan(content string) []ScanIssue {
	var issues []ScanIssue
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for _, p := range maliciousPatterns {
			if p.Pattern.MatchString(line) {
				issues = append(issues, ScanIssue{
					Type:   "security",
					Line:   lineNum + 1,
					Detail: fmt.Sprintf("Potential %s detected", p.Name),
				})
			}
		}
	}

	return issues
}

// PromptInjectionScan 检测 prompt 注入攻击
func PromptInjectionScan(content string) []ScanIssue {
	detector := security.NewPromptInjectionDetector()
	result := detector.Detect(content)

	if result.IsSafe {
		return []ScanIssue{}
	}

	issues := []ScanIssue{}
	for _, finding := range result.Findings {
		issues = append(issues, ScanIssue{
			Type:   "prompt_injection",
			Line:   finding.Line,
			Detail: fmt.Sprintf("%s: %s (context: %s)", finding.Pattern, finding.Description, finding.Context),
		})
	}

	return issues
}
