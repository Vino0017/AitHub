package review

import (
	"fmt"
	"regexp"
	"strings"
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
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "env_var:") || strings.HasPrefix(trimmed, "obtain_url:") {
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
