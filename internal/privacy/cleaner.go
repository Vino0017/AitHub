package privacy

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cleaner handles privacy-sensitive data cleaning.
type Cleaner struct {
	pool *pgxpool.Pool
}

func NewCleaner(pool *pgxpool.Pool) *Cleaner {
	return &Cleaner{pool: pool}
}

// CleaningPattern defines a pattern to detect and clean.
type CleaningPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
	Severity    string // critical | high | medium | low
}

var cleaningPatterns = []CleaningPattern{
	{
		Name:        "AWS Access Key",
		Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		Replacement: "<AWS_ACCESS_KEY>",
		Severity:    "critical",
	},
	{
		Name:        "GitHub Token",
		Pattern:     regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
		Replacement: "<GITHUB_TOKEN>",
		Severity:    "critical",
	},
	{
		Name:        "OpenAI API Key",
		Pattern:     regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}T3BlbkFJ[a-zA-Z0-9]{20,}`),
		Replacement: "<OPENAI_API_KEY>",
		Severity:    "critical",
	},
	{
		Name:        "OpenAI Project Key",
		Pattern:     regexp.MustCompile(`sk-proj-[a-zA-Z0-9]{40,}`),
		Replacement: "<OPENAI_PROJECT_KEY>",
		Severity:    "critical",
	},
	{
		Name:        "Anthropic Key",
		Pattern:     regexp.MustCompile(`sk-ant-[a-zA-Z0-9-]{40,}`),
		Replacement: "<ANTHROPIC_API_KEY>",
		Severity:    "critical",
	},
	{
		Name:        "Email Address",
		Pattern:     regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		Replacement: "<EMAIL>",
		Severity:    "high",
	},
	{
		Name:        "IPv4 Address",
		Pattern:     regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`),
		Replacement: "<IP_ADDRESS>",
		Severity:    "medium",
	},
	{
		Name:        "Private Key",
		Pattern:     regexp.MustCompile(`-----BEGIN (RSA|EC|DSA|OPENSSH|PGP) PRIVATE KEY-----`),
		Replacement: "<PRIVATE_KEY>",
		Severity:    "critical",
	},
	{
		Name:        "Bearer Token",
		Pattern:     regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-.]{20,}`),
		Replacement: "Bearer <TOKEN>",
		Severity:    "critical",
	},
	{
		Name:        "Connection String",
		Pattern:     regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^\s'"]+:[^\s'"]+@`),
		Replacement: "$1://<USER>:<PASSWORD>@",
		Severity:    "critical",
	},
}

// CleanContent applies all cleaning patterns to content.
func (c *Cleaner) CleanContent(content string) (string, CleaningReport) {
	report := CleaningReport{
		OriginalLength: len(content),
		Findings:       []Finding{},
	}

	cleaned := content
	lines := strings.Split(content, "\n")

	for _, pattern := range cleaningPatterns {
		for lineNum, line := range lines {
			// Skip YAML frontmatter field definitions
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "env_var:") || strings.HasPrefix(trimmed, "obtain_url:") {
				continue
			}

			matches := pattern.Pattern.FindAllString(line, -1)
			if len(matches) > 0 {
				for _, match := range matches {
					report.Findings = append(report.Findings, Finding{
						Type:     pattern.Name,
						Line:     lineNum + 1,
						Severity: pattern.Severity,
						Original: truncate(match, 20),
					})
				}
				// Replace in the full content
				cleaned = pattern.Pattern.ReplaceAllString(cleaned, pattern.Replacement)
			}
		}
	}

	report.CleanedLength = len(cleaned)
	report.ItemsCleaned = len(report.Findings)

	return cleaned, report
}

// ForceCleanRevision forcibly cleans a revision's content.
func (c *Cleaner) ForceCleanRevision(ctx context.Context, revisionID uuid.UUID) (CleaningReport, error) {
	var content string
	err := c.pool.QueryRow(ctx,
		`SELECT content FROM revisions WHERE id = $1`, revisionID).Scan(&content)
	if err != nil {
		return CleaningReport{}, err
	}

	cleaned, report := c.CleanContent(content)

	if report.ItemsCleaned > 0 {
		// Update the revision with cleaned content
		_, err = c.pool.Exec(ctx,
			`UPDATE revisions SET content = $2 WHERE id = $1`, revisionID, cleaned)
		if err != nil {
			return report, err
		}
	}

	return report, nil
}

// ScanAllRevisions scans all revisions for privacy issues.
func (c *Cleaner) ScanAllRevisions(ctx context.Context) ([]RevisionScanResult, error) {
	rows, err := c.pool.Query(ctx,
		`SELECT r.id, s.id, n.name, s.name, r.version, r.content
		 FROM revisions r
		 JOIN skills s ON r.skill_id = s.id
		 JOIN namespaces n ON s.namespace_id = n.id
		 WHERE r.review_status = 'approved'
		 ORDER BY r.created_at DESC
		 LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []RevisionScanResult{}
	for rows.Next() {
		var revID, skillID uuid.UUID
		var ns, skillName, version, content string
		rows.Scan(&revID, &skillID, &ns, &skillName, &version, &content)

		_, report := c.CleanContent(content)
		if report.ItemsCleaned > 0 {
			results = append(results, RevisionScanResult{
				RevisionID: revID,
				SkillID:    skillID,
				FullName:   ns + "/" + skillName,
				Version:    version,
				Report:     report,
			})
		}
	}

	return results, nil
}

// CleaningReport describes what was cleaned.
type CleaningReport struct {
	OriginalLength int       `json:"original_length"`
	CleanedLength  int       `json:"cleaned_length"`
	ItemsCleaned   int       `json:"items_cleaned"`
	Findings       []Finding `json:"findings"`
}

type Finding struct {
	Type     string `json:"type"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Original string `json:"original"`
}

type RevisionScanResult struct {
	RevisionID uuid.UUID      `json:"revision_id"`
	SkillID    uuid.UUID      `json:"skill_id"`
	FullName   string         `json:"full_name"`
	Version    string         `json:"version"`
	Report     CleaningReport `json:"report"`
}

func (cr *CleaningReport) ToJSON() ([]byte, error) {
	return json.Marshal(cr)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
