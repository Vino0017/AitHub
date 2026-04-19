package privacy

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestNewCleaner tests cleaner creation
func TestNewCleaner(t *testing.T) {
	pool := &pgxpool.Pool{}

	cleaner := NewCleaner(pool)

	if cleaner == nil {
		t.Fatal("Expected non-nil cleaner")
	}
	if cleaner.pool != pool {
		t.Error("Expected pool to be set")
	}
}

// TestCleanContent_AWSAccessKey tests AWS access key detection
func TestCleanContent_AWSAccessKey(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "<AWS_ACCESS_KEY>") {
		t.Error("Expected AWS access key to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
	if len(report.Findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(report.Findings))
	}
	if report.Findings[0].Severity != "critical" {
		t.Errorf("Expected critical severity, got %s", report.Findings[0].Severity)
	}
}

// TestCleanContent_GitHubToken tests GitHub token detection
func TestCleanContent_GitHubToken(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz"
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "<GITHUB_TOKEN>") {
		t.Error("Expected GitHub token to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
}

// TestCleanContent_OpenAIProjectKey tests OpenAI project key detection
func TestCleanContent_OpenAIProjectKey(t *testing.T) {
	cleaner := NewCleaner(nil)

	// OpenAI project keys are typically 40+ characters
	content := "OPENAI_API_KEY=sk-proj-" + strings.Repeat("a", 50)
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "<OPENAI_PROJECT_KEY>") {
		t.Error("Expected OpenAI project key to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
}

// TestCleanContent_AnthropicKey tests Anthropic key detection
func TestCleanContent_AnthropicKey(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "ANTHROPIC_API_KEY=sk-ant-api03-abcdefghijklmnopqrstuvwxyz1234567890"
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "<ANTHROPIC_API_KEY>") {
		t.Error("Expected Anthropic key to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
}

// TestCleanContent_EmailAddress tests email address detection
func TestCleanContent_EmailAddress(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "Contact me at user@example.com for more info"
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "<EMAIL>") {
		t.Error("Expected email to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
	if report.Findings[0].Severity != "high" {
		t.Errorf("Expected high severity, got %s", report.Findings[0].Severity)
	}
}

// TestCleanContent_IPv4Address tests IPv4 address detection
func TestCleanContent_IPv4Address(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "Server IP: 192.168.1.100"
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "<IP_ADDRESS>") {
		t.Error("Expected IP address to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
	if report.Findings[0].Severity != "medium" {
		t.Errorf("Expected medium severity, got %s", report.Findings[0].Severity)
	}
}

// TestCleanContent_PrivateKey tests private key detection
func TestCleanContent_PrivateKey(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA..."
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "<PRIVATE_KEY>") {
		t.Error("Expected private key to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
	if report.Findings[0].Severity != "critical" {
		t.Errorf("Expected critical severity, got %s", report.Findings[0].Severity)
	}
}

// TestCleanContent_BearerToken tests bearer token detection
func TestCleanContent_BearerToken(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	cleaned, report := cleaner.CleanContent(content)

	if !strings.Contains(cleaned, "Bearer <TOKEN>") {
		t.Error("Expected bearer token to be cleaned")
	}
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}
}

// TestCleanContent_ConnectionString tests connection string detection
func TestCleanContent_ConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			"PostgreSQL",
			"DATABASE_URL=postgres://user:password@localhost:5432/db",
			"postgres://<USER>:<PASSWORD>@",
		},
		{
			"MySQL",
			"DB_URL=mysql://root:secret@localhost:3306/mydb",
			"mysql://<USER>:<PASSWORD>@",
		},
		{
			"MongoDB",
			"MONGO_URI=mongodb://admin:pass123@localhost:27017/db",
			"mongodb://<USER>:<PASSWORD>@",
		},
	}

	cleaner := NewCleaner(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned, report := cleaner.CleanContent(tt.content)

			if !strings.Contains(cleaned, tt.expected) {
				t.Errorf("Expected cleaned content to contain '%s', got '%s'", tt.expected, cleaned)
			}
			if report.ItemsCleaned != 1 {
				t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
			}
			if report.Findings[0].Severity != "critical" {
				t.Errorf("Expected critical severity, got %s", report.Findings[0].Severity)
			}
		})
	}
}

// TestCleanContent_MultipleFindings tests multiple findings in one content
func TestCleanContent_MultipleFindings(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := `
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz
Contact: user@example.com
Server: 192.168.1.100
`
	cleaned, report := cleaner.CleanContent(content)

	if report.ItemsCleaned != 4 {
		t.Errorf("Expected 4 items cleaned, got %d", report.ItemsCleaned)
	}
	if len(report.Findings) != 4 {
		t.Errorf("Expected 4 findings, got %d", len(report.Findings))
	}
	if !strings.Contains(cleaned, "<AWS_ACCESS_KEY>") {
		t.Error("Expected AWS key to be cleaned")
	}
	if !strings.Contains(cleaned, "<GITHUB_TOKEN>") {
		t.Error("Expected GitHub token to be cleaned")
	}
	if !strings.Contains(cleaned, "<EMAIL>") {
		t.Error("Expected email to be cleaned")
	}
	if !strings.Contains(cleaned, "<IP_ADDRESS>") {
		t.Error("Expected IP to be cleaned")
	}
}

// TestCleanContent_SkipYAMLFrontmatter tests skipping YAML frontmatter fields
func TestCleanContent_SkipYAMLFrontmatter(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := `---
name: test-skill
env_var: OPENAI_API_KEY
obtain_url: https://platform.openai.com/api-keys
---

# Test Skill

Use OPENAI_API_KEY=sk-proj-` + strings.Repeat("a", 50) + ` to configure.
`
	cleaned, report := cleaner.CleanContent(content)

	// Should only clean the actual key in the body, not the field definitions
	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned (only the actual key), got %d", report.ItemsCleaned)
	}
	// The env_var and obtain_url lines should remain unchanged
	if !strings.Contains(cleaned, "env_var: OPENAI_API_KEY") {
		t.Error("Expected env_var field to remain unchanged")
	}
	if !strings.Contains(cleaned, "obtain_url: https://platform.openai.com/api-keys") {
		t.Error("Expected obtain_url field to remain unchanged")
	}
	// But the actual key should be cleaned
	if !strings.Contains(cleaned, "<OPENAI_PROJECT_KEY>") {
		t.Error("Expected actual key to be cleaned")
	}
}

// TestCleanContent_CleanContent tests clean content (no findings)
func TestCleanContent_CleanContent(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := "This is clean content with no secrets or PII."
	cleaned, report := cleaner.CleanContent(content)

	if cleaned != content {
		t.Error("Expected content to remain unchanged")
	}
	if report.ItemsCleaned != 0 {
		t.Errorf("Expected 0 items cleaned, got %d", report.ItemsCleaned)
	}
	if len(report.Findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(report.Findings))
	}
}

// TestCleanContent_LineNumbers tests correct line number reporting
func TestCleanContent_LineNumbers(t *testing.T) {
	cleaner := NewCleaner(nil)

	content := `Line 1: clean
Line 2: AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
Line 3: clean
Line 4: user@example.com
Line 5: clean`

	_, report := cleaner.CleanContent(content)

	if len(report.Findings) != 2 {
		t.Fatalf("Expected 2 findings, got %d", len(report.Findings))
	}

	// Check line numbers
	if report.Findings[0].Line != 2 {
		t.Errorf("Expected finding on line 2, got line %d", report.Findings[0].Line)
	}
	if report.Findings[1].Line != 4 {
		t.Errorf("Expected finding on line 4, got line %d", report.Findings[1].Line)
	}
}

// TestCleanContent_Truncation tests original value truncation
func TestCleanContent_Truncation(t *testing.T) {
	cleaner := NewCleaner(nil)

	longKey := "sk-proj-" + strings.Repeat("a", 100)
	content := "OPENAI_API_KEY=" + longKey

	_, report := cleaner.CleanContent(content)

	if len(report.Findings) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(report.Findings))
	}

	// Original should be truncated to 20 chars + "..."
	if len(report.Findings[0].Original) > 23 {
		t.Errorf("Expected truncated original (max 23 chars), got %d chars", len(report.Findings[0].Original))
	}
	if !strings.HasSuffix(report.Findings[0].Original, "...") {
		t.Error("Expected truncated original to end with '...'")
	}
}

// TestCleaningReport_ToJSON tests JSON marshaling
func TestCleaningReport_ToJSON(t *testing.T) {
	report := CleaningReport{
		OriginalLength: 100,
		CleanedLength:  80,
		ItemsCleaned:   2,
		Findings: []Finding{
			{Type: "AWS Access Key", Line: 1, Severity: "critical", Original: "AKIA..."},
			{Type: "Email Address", Line: 2, Severity: "high", Original: "user@example.com"},
		},
	}

	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON")
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "original_length") {
		t.Error("Expected JSON to contain 'original_length'")
	}
	if !strings.Contains(jsonStr, "items_cleaned") {
		t.Error("Expected JSON to contain 'items_cleaned'")
	}
	if !strings.Contains(jsonStr, "findings") {
		t.Error("Expected JSON to contain 'findings'")
	}
}

// TestForceCleanRevision tests revision cleaning
func TestForceCleanRevision(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	cleaner := NewCleaner(pool)

	// Create test revision with secrets
	revisionID := uuid.New()
	skillID := uuid.New()
	content := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"

	_, err := pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, content) VALUES ($1, $2, $3)`,
		revisionID, skillID, content)
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Clean the revision
	report, err := cleaner.ForceCleanRevision(context.Background(), revisionID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", report.ItemsCleaned)
	}

	// Verify content was updated
	var cleanedContent string
	err = pool.QueryRow(context.Background(),
		`SELECT content FROM revisions WHERE id = $1`, revisionID).Scan(&cleanedContent)
	if err != nil {
		t.Fatalf("Failed to get cleaned content: %v", err)
	}

	if !strings.Contains(cleanedContent, "<AWS_ACCESS_KEY>") {
		t.Error("Expected content to be cleaned in database")
	}
}

// TestScanAllRevisions tests scanning all revisions
func TestScanAllRevisions(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	cleaner := NewCleaner(pool)

	// Create test data
	nsID := uuid.New()
	skillID := uuid.New()
	revisionID := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO namespaces (id, name, type, email) VALUES ($1, $2, 'personal', $3)`,
		nsID, "test-user", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO skills (id, namespace_id, name) VALUES ($1, $2, $3)`,
		skillID, nsID, "test-skill")
	if err != nil {
		t.Fatalf("Failed to create skill: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, version, content, review_status) VALUES ($1, $2, $3, $4, $5)`,
		revisionID, skillID, "1.0.0", "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE", "approved")
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Scan all revisions
	results, err := cleaner.ScanAllRevisions(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].RevisionID != revisionID {
		t.Error("Expected matching revision ID")
	}
	if results[0].Report.ItemsCleaned != 1 {
		t.Errorf("Expected 1 item cleaned, got %d", results[0].Report.ItemsCleaned)
	}
}

// TestTruncate tests truncate helper function
func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"Short string", "hello", 10, "hello"},
		{"Exact length", "hello", 5, "hello"},
		{"Long string", "hello world", 5, "hello..."},
		{"Very long", strings.Repeat("a", 100), 20, strings.Repeat("a", 20) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// --- Test Helpers ---

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test DB: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("Skipping test: cannot ping test DB: %v", err)
	}

	cleanup := func() {
		pool.Exec(context.Background(), "TRUNCATE revisions, skills, namespaces CASCADE")
		pool.Close()
	}

	return pool, cleanup
}
