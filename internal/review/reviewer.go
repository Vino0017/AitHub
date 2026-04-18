package review

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/llm"
)

const maxRetries = 3

// Reviewer handles the two-layer review process.
type Reviewer struct {
	pool    *pgxpool.Pool
	llm     *llm.Client
	enabled bool
}

func NewReviewer(pool *pgxpool.Pool, llmClient *llm.Client, enabled bool) *Reviewer {
	return &Reviewer{pool: pool, llm: llmClient, enabled: enabled}
}

// ReviewResult holds the outcome of reviewing a revision.
type ReviewResult struct {
	Status   string      `json:"status"` // approved | revision_requested | rejected
	Issues   []ScanIssue `json:"issues,omitempty"`
	Feedback string      `json:"feedback,omitempty"`
}

// Review performs the two-layer review on a revision.
func (rv *Reviewer) Review(ctx context.Context, revisionID uuid.UUID) {
	var content string
	var retryCount int
	var skillID uuid.UUID

	err := rv.pool.QueryRow(ctx,
		`SELECT r.content, r.review_retry_count, r.skill_id FROM revisions r WHERE r.id = $1`,
		revisionID).Scan(&content, &retryCount, &skillID)
	if err != nil {
		log.Printf("review: failed to get revision %s: %v", revisionID, err)
		return
	}

	// Circuit breaker: max retries
	if retryCount >= maxRetries {
		rv.setResult(ctx, revisionID, skillID, "rejected", []ScanIssue{
			{Type: "system", Detail: "Maximum review retries exceeded (3). Submission rejected."},
		})
		return
	}

	// Increment retry count
	rv.pool.Exec(ctx,
		`UPDATE revisions SET review_retry_count = review_retry_count + 1 WHERE id = $1`, revisionID)

	// Layer 1: Regex scan (< 1ms, zero cost)
	secretIssues := RegexScan(content)
	securityIssues := SecurityScan(content)

	if len(securityIssues) > 0 {
		// Malicious content → reject
		allIssues := append(securityIssues, secretIssues...)
		rv.setResult(ctx, revisionID, skillID, "rejected", allIssues)
		return
	}

	if len(secretIssues) > 0 {
		// Secrets found → revision_requested (not reject, let agent fix)
		rv.setResult(ctx, revisionID, skillID, "revision_requested", secretIssues)
		return
	}

	// Layer 2: LLM deep review (only if regex passed)
	if rv.enabled && rv.llm != nil {
		result := rv.llmReview(ctx, content)
		rv.setResult(ctx, revisionID, skillID, result.Status, result.Issues)
		return
	}

	// No LLM configured → auto-approve
	rv.setResult(ctx, revisionID, skillID, "approved", nil)
}

func (rv *Reviewer) llmReview(ctx context.Context, content string) ReviewResult {
	prompt := fmt.Sprintf(`You are a security reviewer for SkillHub, an AI skill registry.
Review this SKILL.md submission for:
1. Malicious commands (rm -rf, reverse shells, data exfiltration, crypto mining)
2. Privacy leaks (real API keys, passwords, email addresses, real names, IP addresses)
3. Format quality (does it have clear, actionable instructions?)
4. Content quality (is it a real useful skill, not spam or placeholder?)

If you find issues, provide specific fixes the AI can apply.

Respond in JSON format:
{
  "status": "approved|revision_requested|rejected",
  "issues": [
    {
      "type": "privacy|security|format|quality",
      "detail": "...",
      "suggested_fix": "Replace line 42: 'sk-proj-abc123' with '<API_KEY>'"
    }
  ]
}

SKILL.md content:
%s`, content)

	resp, err := rv.llm.Complete(ctx, prompt)
	if err != nil {
		log.Printf("review: LLM error: %v, auto-approving", err)
		return ReviewResult{Status: "approved"}
	}

	var result ReviewResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		log.Printf("review: failed to parse LLM response: %v, auto-approving", err)
		return ReviewResult{Status: "approved"}
	}

	return result
}

func (rv *Reviewer) setResult(ctx context.Context, revisionID, skillID uuid.UUID, status string, issues []ScanIssue) {
	feedback, _ := json.Marshal(map[string]interface{}{
		"issues": issues,
	})
	result, _ := json.Marshal(map[string]string{"status": status})

	rv.pool.Exec(ctx,
		`UPDATE revisions SET review_status = $2, review_feedback = $3, review_result = $4 WHERE id = $1`,
		revisionID, status, feedback, result)

	if status == "approved" {
		// Update skill's latest_version
		var version string
		rv.pool.QueryRow(ctx, `SELECT version FROM revisions WHERE id = $1`, revisionID).Scan(&version)
		rv.pool.Exec(ctx,
			`UPDATE skills SET latest_version = $2, updated_at = NOW() WHERE id = $1`, skillID, version)
	}

	log.Printf("review: revision %s → %s (%d issues)", revisionID, status, len(issues))
}
