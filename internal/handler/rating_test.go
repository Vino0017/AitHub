package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRating_ValidSubmission tests successful rating submission
func TestRating_ValidSubmission(t *testing.T) {
	t.Skip("Requires full database setup with migrations")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	_, _ = setupTestSkill(t, pool, nsID)

	handler := NewRatingHandler(pool)

	rating := map[string]interface{}{
		"score":           9,
		"outcome":         "success",
		"task_type":       "code-review",
		"model_used":      "claude-opus-4",
		"tokens_consumed": 1200,
	}
	body, _ := json.Marshal(rating)

	req := httptest.NewRequest("POST", "/v1/skills/test-user/test-skill/ratings", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()

	handler.Submit(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRating_InvalidScore tests score validation
func TestRating_InvalidScore(t *testing.T) {
	t.Skip("Requires full database setup with migrations")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	_, _ = setupTestSkill(t, pool, nsID)

	handler := NewRatingHandler(pool)

	tests := []struct {
		name  string
		score int
	}{
		{"score too low", -1},
		{"score too high", 11},
		{"score zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rating := map[string]interface{}{
				"score":   tt.score,
				"outcome": "success",
			}
			body, _ := json.Marshal(rating)

			req := httptest.NewRequest("POST", "/v1/skills/test-user/test-skill/ratings", bytes.NewReader(body))
			req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
			w := httptest.NewRecorder()

			handler.Submit(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for score %d, got %d", tt.score, w.Code)
			}
		})
	}
}

// TestRating_InvalidOutcome tests outcome validation
func TestRating_InvalidOutcome(t *testing.T) {
	t.Skip("Requires full database setup with migrations")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	_, _ = setupTestSkill(t, pool, nsID)

	handler := NewRatingHandler(pool)

	rating := map[string]interface{}{
		"score":   8,
		"outcome": "invalid_outcome",
	}
	body, _ := json.Marshal(rating)

	req := httptest.NewRequest("POST", "/v1/skills/test-user/test-skill/ratings", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()

	handler.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// TestRating_UpdateExisting tests upsert behavior
func TestRating_UpdateExisting(t *testing.T) {
	t.Skip("Requires full database setup with migrations")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)
	_, revisionID := setupTestSkill(t, pool, nsID)

	handler := NewRatingHandler(pool)

	// First rating
	rating1 := map[string]interface{}{
		"score":   7,
		"outcome": "success",
	}
	body1, _ := json.Marshal(rating1)
	req1 := httptest.NewRequest("POST", "/v1/skills/test-user/test-skill/ratings", bytes.NewReader(body1))
	req1 = req1.WithContext(contextWithAuth(req1.Context(), nsID, tokenID))
	w1 := httptest.NewRecorder()
	handler.Submit(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("First rating failed: %d", w1.Code)
	}

	// Second rating (should update)
	rating2 := map[string]interface{}{
		"score":   9,
		"outcome": "success",
	}
	body2, _ := json.Marshal(rating2)
	req2 := httptest.NewRequest("POST", "/v1/skills/test-user/test-skill/ratings", bytes.NewReader(body2))
	req2 = req2.WithContext(contextWithAuth(req2.Context(), nsID, tokenID))
	w2 := httptest.NewRecorder()
	handler.Submit(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("Second rating failed: %d", w2.Code)
	}

	// Verify only one rating exists
	var count int
	pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM ratings WHERE revision_id = $1 AND token_id = $2",
		revisionID, tokenID).Scan(&count)

	if count != 1 {
		t.Errorf("Expected 1 rating (upsert), got %d", count)
	}

	// Verify score was updated
	var score int
	pool.QueryRow(context.Background(),
		"SELECT score FROM ratings WHERE revision_id = $1 AND token_id = $2",
		revisionID, tokenID).Scan(&score)

	if score != 9 {
		t.Errorf("Expected score 9, got %d", score)
	}
}

// TestRating_NonexistentSkill tests rating a skill that doesn't exist
func TestRating_NonexistentSkill(t *testing.T) {
	t.Skip("Requires full database setup with migrations")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	nsID, tokenID := setupTestNamespace(t, pool)

	handler := NewRatingHandler(pool)

	rating := map[string]interface{}{
		"score":   8,
		"outcome": "success",
	}
	body, _ := json.Marshal(rating)

	req := httptest.NewRequest("POST", "/v1/skills/nonexistent/skill/ratings", bytes.NewReader(body))
	req = req.WithContext(contextWithAuth(req.Context(), nsID, tokenID))
	w := httptest.NewRecorder()

	handler.Submit(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

// --- Test Helpers are in testing.go ---
