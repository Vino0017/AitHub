package review

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/llm"
)

// TestNewReviewer tests reviewer creation
func TestNewReviewer(t *testing.T) {
	pool := &pgxpool.Pool{}
	llmClient := &llm.Client{}

	reviewer := NewReviewer(pool, llmClient, true)

	if reviewer == nil {
		t.Fatal("Expected non-nil reviewer")
	}
	if reviewer.pool != pool {
		t.Error("Expected pool to be set")
	}
	if reviewer.llm != llmClient {
		t.Error("Expected llm client to be set")
	}
	if !reviewer.enabled {
		t.Error("Expected enabled to be true")
	}
}

// TestNewReviewer_Disabled tests reviewer with LLM disabled
func TestNewReviewer_Disabled(t *testing.T) {
	pool := &pgxpool.Pool{}

	reviewer := NewReviewer(pool, nil, false)

	if reviewer.enabled {
		t.Error("Expected enabled to be false")
	}
	if reviewer.llm != nil {
		t.Error("Expected llm to be nil")
	}
}

// TestReview_MaliciousContent tests rejection of malicious content
func TestReview_MaliciousContent(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	reviewer := NewReviewer(pool, nil, false)

	// Create test revision with malicious content
	revisionID := uuid.New()
	skillID := uuid.New()
	maliciousContent := "---\nname: test\nversion: 1.0.0\n---\n\nrm -rf / --no-preserve-root"

	_, err := pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, content, review_retry_count) VALUES ($1, $2, $3, 0)`,
		revisionID, skillID, maliciousContent)
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Run review
	reviewer.Review(context.Background(), revisionID)

	// Check result
	var status string
	err = pool.QueryRow(context.Background(),
		`SELECT review_status FROM revisions WHERE id = $1`, revisionID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to get review status: %v", err)
	}

	if status != "rejected" {
		t.Errorf("Expected status 'rejected', got '%s'", status)
	}
}

// TestReview_SecretsFound tests revision_requested for secrets
func TestReview_SecretsFound(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	reviewer := NewReviewer(pool, nil, false)

	// Create test revision with secrets
	revisionID := uuid.New()
	skillID := uuid.New()
	contentWithSecrets := "---\nname: test\nversion: 1.0.0\n---\n\nAPI_KEY=sk-proj-abc123def456"

	_, err := pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, content, review_retry_count) VALUES ($1, $2, $3, 0)`,
		revisionID, skillID, contentWithSecrets)
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Run review
	reviewer.Review(context.Background(), revisionID)

	// Check result
	var status string
	err = pool.QueryRow(context.Background(),
		`SELECT review_status FROM revisions WHERE id = $1`, revisionID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to get review status: %v", err)
	}

	if status != "revision_requested" {
		t.Errorf("Expected status 'revision_requested', got '%s'", status)
	}
}

// TestReview_CleanContent tests auto-approval of clean content
func TestReview_CleanContent(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	reviewer := NewReviewer(pool, nil, false)

	// Create test revision with clean content
	revisionID := uuid.New()
	skillID := uuid.New()
	cleanContent := "---\nname: test-skill\nversion: 1.0.0\nframework: gstack\ntags: [test]\ndescription: test\n---\n\n# Test Skill\n\nThis is a test skill."

	_, err := pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, content, review_retry_count) VALUES ($1, $2, $3, 0)`,
		revisionID, skillID, cleanContent)
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Run review
	reviewer.Review(context.Background(), revisionID)

	// Check result
	var status string
	err = pool.QueryRow(context.Background(),
		`SELECT review_status FROM revisions WHERE id = $1`, revisionID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to get review status: %v", err)
	}

	if status != "approved" {
		t.Errorf("Expected status 'approved', got '%s'", status)
	}
}

// TestReview_MaxRetries tests circuit breaker
func TestReview_MaxRetries(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	reviewer := NewReviewer(pool, nil, false)

	// Create test revision with max retries
	revisionID := uuid.New()
	skillID := uuid.New()
	content := "---\nname: test\nversion: 1.0.0\n---\n\nTest content"

	_, err := pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, content, review_retry_count) VALUES ($1, $2, $3, 3)`,
		revisionID, skillID, content)
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Run review
	reviewer.Review(context.Background(), revisionID)

	// Check result
	var status string
	var feedback string
	err = pool.QueryRow(context.Background(),
		`SELECT review_status, review_feedback FROM revisions WHERE id = $1`, revisionID).Scan(&status, &feedback)
	if err != nil {
		t.Fatalf("Failed to get review status: %v", err)
	}

	if status != "rejected" {
		t.Errorf("Expected status 'rejected', got '%s'", status)
	}

	if feedback == "" {
		t.Error("Expected feedback about max retries")
	}
}

// TestLLMReview_NotConfigured tests behavior when LLM is not configured
func TestLLMReview_NotConfigured(t *testing.T) {
	// When LLM is not configured, reviewer should auto-approve
	reviewer := NewReviewer(nil, nil, false)

	// Verify reviewer is not enabled
	if reviewer.enabled {
		t.Error("Expected reviewer to be disabled")
	}
	if reviewer.llm != nil {
		t.Error("Expected llm to be nil")
	}
}

// TestReviewer_LLMEnabled tests reviewer with LLM enabled
func TestReviewer_LLMEnabled(t *testing.T) {
	llmClient := &llm.Client{}
	reviewer := NewReviewer(nil, llmClient, true)

	// Verify reviewer is enabled
	if !reviewer.enabled {
		t.Error("Expected reviewer to be enabled")
	}
	if reviewer.llm == nil {
		t.Error("Expected llm to be set")
	}
}

// TestSetResult tests result persistence
func TestSetResult(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	reviewer := NewReviewer(pool, nil, false)

	revisionID := uuid.New()
	skillID := uuid.New()

	// Create test revision
	_, err := pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, content, version) VALUES ($1, $2, $3, $4)`,
		revisionID, skillID, "test content", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Create test skill
	_, err = pool.Exec(context.Background(),
		`INSERT INTO skills (id, namespace_id, name) VALUES ($1, $2, $3)`,
		skillID, uuid.New(), "test-skill")
	if err != nil {
		t.Fatalf("Failed to create skill: %v", err)
	}

	// Set result
	issues := []ScanIssue{{Type: "test", Detail: "test issue"}}
	reviewer.setResult(context.Background(), revisionID, skillID, "approved", issues)

	// Verify result
	var status string
	var feedback string
	err = pool.QueryRow(context.Background(),
		`SELECT review_status, review_feedback FROM revisions WHERE id = $1`, revisionID).Scan(&status, &feedback)
	if err != nil {
		t.Fatalf("Failed to get review result: %v", err)
	}

	if status != "approved" {
		t.Errorf("Expected status 'approved', got '%s'", status)
	}

	// Verify skill was updated
	var latestVersion string
	err = pool.QueryRow(context.Background(),
		`SELECT latest_version FROM skills WHERE id = $1`, skillID).Scan(&latestVersion)
	if err != nil {
		t.Fatalf("Failed to get skill version: %v", err)
	}

	if latestVersion != "1.0.0" {
		t.Errorf("Expected latest_version '1.0.0', got '%s'", latestVersion)
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
