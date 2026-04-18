package handler

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type RatingHandler struct {
	pool *pgxpool.Pool
}

func NewRatingHandler(pool *pgxpool.Pool) *RatingHandler {
	return &RatingHandler{pool: pool}
}

// Submit handles POST /v1/skills/{namespace}/{name}/ratings
func (h *RatingHandler) Submit(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Score          int    `json:"score"`
		Outcome        string `json:"outcome"`
		TaskType       string `json:"task_type"`
		ModelUsed      string `json:"model_used"`
		TokensConsumed int    `json:"tokens_consumed"`
		FailureReason  string `json:"failure_reason"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body", "")
		return
	}

	if req.Score < 1 || req.Score > 10 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_score", "Score must be between 1 and 10", "")
		return
	}
	if req.Outcome == "" {
		req.Outcome = "success"
	}
	if req.Outcome != "success" && req.Outcome != "partial" && req.Outcome != "failure" {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_outcome", "Outcome must be success, partial, or failure", "")
		return
	}

	// Find skill (with visibility check)
	skillID, ok := checkSkillAccess(h.pool, w, r)
	if !ok {
		return
	}

	// Find latest approved revision
	var revisionID interface{}
	err := h.pool.QueryRow(r.Context(),
		`SELECT id FROM revisions WHERE skill_id = $1 AND review_status = 'approved'
		 ORDER BY created_at DESC LIMIT 1`, skillID).Scan(&revisionID)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "no_approved_revision", "No approved revision to rate", "")
		return
	}

	tokenID := middleware.GetTokenID(r.Context())

	// Upsert rating
	var ratingID interface{}
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO ratings (skill_id, revision_id, token_id, score, outcome, task_type, model_used, tokens_consumed, failure_reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (revision_id, token_id) DO UPDATE SET
		   score = EXCLUDED.score, outcome = EXCLUDED.outcome,
		   task_type = EXCLUDED.task_type, model_used = EXCLUDED.model_used,
		   tokens_consumed = EXCLUDED.tokens_consumed, failure_reason = EXCLUDED.failure_reason,
		   updated_at = NOW()
		 RETURNING id`,
		skillID, revisionID, tokenID, req.Score, req.Outcome,
		req.TaskType, req.ModelUsed, req.TokensConsumed, req.FailureReason).Scan(&ratingID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to submit rating", "")
		return
	}

	// Recalculate skill rating (only from registered tokens on latest revision)
	go h.refreshSkillRating(skillID, revisionID)

	helpers.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      ratingID,
		"status":  "recorded",
		"upsert":  true,
	})
}

func (h *RatingHandler) refreshSkillRating(skillID, revisionID interface{}) {
	ctx := context.Background()
	var count int
	var avgScore, successRate float64

	// Only count ratings from registered (non-anonymous) tokens
	h.pool.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(AVG(r.score), 0), 
		        COALESCE(SUM(CASE WHEN r.outcome = 'success' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0)
		 FROM ratings r JOIN tokens t ON r.token_id = t.id
		 WHERE r.revision_id = $1 AND t.namespace_id IS NOT NULL`,
		revisionID).Scan(&count, &avgScore, &successRate)

	// Bayesian average: (C*m + sum) / (C + n), C=5, m=6.0
	bayesian := 0.0
	if count > 0 {
		bayesian = (5.0*6.0 + avgScore*float64(count)) / (5.0 + float64(count))
	}

	h.pool.Exec(ctx,
		`UPDATE skills SET avg_rating = $2, rating_count = $3, outcome_success_rate = $4, updated_at = NOW() WHERE id = $1`,
		skillID, bayesian, count, successRate)
}
