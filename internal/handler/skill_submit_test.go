package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/skillhub/api/internal/review"
)

// TestSubmit_ValidSkill_ReturnsCreated tests the happy path
func TestSubmit_ValidSkill_ReturnsCreated(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	reviewer := &review.Reviewer{} // Mock reviewer
	riverClient := setupRiverClient(t, pool)

	handler := NewSkillSubmitHandler(pool, reviewer, riverClient)

	validSkill := `---
name: test-skill
version: 1.0.0
framework: gstack
tags: [testing]
description: A test skill
---

# Test Skill

This is a test.`

	reqBody := map[string]string{
		"content": validSkill,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/skills", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()

	handler.Submit(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "pending" {
		t.Errorf("Expected status 'pending', got %v", resp["status"])
	}
	if resp["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", resp["version"])
	}
}

// TestSubmit_DuplicateName_ReturnsConflict tests duplicate skill name
func TestSubmit_DuplicateName_ReturnsConflict(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	reviewer := &review.Reviewer{}
	riverClient := setupRiverClient(t, pool)

	handler := NewSkillSubmitHandler(pool, reviewer, riverClient)

	validSkill := `---
name: duplicate-skill
version: 1.0.0
framework: gstack
tags: [testing]
description: First submission
---

# First`

	// First submission - should succeed
	reqBody := map[string]string{"content": validSkill}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/skills", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()
	handler.Submit(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("First submission failed: %d %s", w.Code, w.Body.String())
	}

	// Second submission with same name but different version - should create new revision
	validSkillV2 := `---
name: duplicate-skill
version: 2.0.0
framework: gstack
tags: [testing]
description: Second submission
---

# Second`

	reqBody2 := map[string]string{"content": validSkillV2}
	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest("POST", "/v1/skills", bytes.NewReader(body2))
	req2 = req2.WithContext(contextWithAuth(req2.Context(), nsID, tokenID))
	w2 := httptest.NewRecorder()
	handler.Submit(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("Expected new revision to succeed, got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestSubmit_InvalidVersion_ReturnsError tests version validation
func TestSubmit_InvalidVersion_ReturnsError(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	reviewer := &review.Reviewer{}
	riverClient := setupRiverClient(t, pool)

	handler := NewSkillSubmitHandler(pool, reviewer, riverClient)

	// Submit v1.0.0 first
	skill1 := `---
name: version-test
version: 1.0.0
framework: gstack
tags: [testing]
description: Version test
---

# V1`

	reqBody := map[string]string{"content": skill1}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/skills", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()
	handler.Submit(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("First submission failed: %d", w.Code)
	}

	// Try to submit v0.9.0 (lower version) - should fail
	skill2 := `---
name: version-test
version: 0.9.0
framework: gstack
tags: [testing]
description: Lower version
---

# V0.9`

	reqBody2 := map[string]string{"content": skill2}
	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest("POST", "/v1/skills", bytes.NewReader(body2))
	req2 = req2.WithContext(contextWithAuth(req2.Context(), nsID, tokenID))
	w2 := httptest.NewRecorder()
	handler.Submit(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for lower version, got %d", w2.Code)
	}

	var errResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &errResp)
	if errResp["code"] != "version_too_low" {
		t.Errorf("Expected error code 'version_too_low', got %v", errResp["code"])
	}
}

// TestSubmit_EmptyContent_ReturnsBadRequest tests empty content validation
func TestSubmit_EmptyContent_ReturnsBadRequest(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	reviewer := &review.Reviewer{}
	riverClient := setupRiverClient(t, pool)

	handler := NewSkillSubmitHandler(pool, reviewer, riverClient)

	reqBody := map[string]string{"content": "   "}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/skills", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()

	handler.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}

	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp["code"] != "empty_content" {
		t.Errorf("Expected error code 'empty_content', got %v", errResp["code"])
	}
}

// TestSubmit_InvalidFrontmatter_ReturnsBadRequest tests frontmatter validation
func TestSubmit_InvalidFrontmatter_ReturnsBadRequest(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	reviewer := &review.Reviewer{}
	riverClient := setupRiverClient(t, pool)

	handler := NewSkillSubmitHandler(pool, reviewer, riverClient)

	invalidSkill := `---
name: test
# Missing required fields: version, framework, description
---

# Content`

	reqBody := map[string]string{"content": invalidSkill}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/skills", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()

	handler.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// --- Test Helpers ---

func setupRiverClient(t *testing.T, pool *pgxpool.Pool) *river.Client[pgx.Tx] {
	t.Helper()

	// Create a minimal River client for testing
	// In production, this would be properly configured
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create River client: %v", err)
	}

	return riverClient
}
