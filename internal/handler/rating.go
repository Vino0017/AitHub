package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/credibility"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type RatingHandler struct {
	pool     *pgxpool.Pool
	analyzer *credibility.Analyzer
}

func NewRatingHandler(pool *pgxpool.Pool) *RatingHandler {
	return &RatingHandler{
		pool:     pool,
		analyzer: credibility.NewAnalyzer(pool),
	}
}

// Submit handles POST /v1/skills/{namespace}/{name}/ratings
func (h *RatingHandler) Submit(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Score           int             `json:"score"`
		Outcome         string          `json:"outcome"`
		TaskType        string          `json:"task_type"`
		ModelUsed       string          `json:"model_used"`
		TokensConsumed  int             `json:"tokens_consumed"`
		FailureReason   string          `json:"failure_reason"`
		ExecutionTimeMs int             `json:"execution_time_ms"`
		ErrorDetails    json.RawMessage `json:"error_details"`
		ContextMetadata json.RawMessage `json:"context_metadata"`
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

	tokenID, ok := middleware.GetTokenID(r.Context()).(uuid.UUID)
	if !ok {
		helpers.WriteError(w, http.StatusUnauthorized, "invalid_token", "Invalid token ID", "")
		return
	}

	// Calculate confidence score
	confidence := h.analyzer.CalculateConfidence(r.Context(), credibility.RatingInput{
		TokenID:         tokenID,
		SkillID:         skillID,
		Score:           req.Score,
		Outcome:         req.Outcome,
		ExecutionTimeMs: req.ExecutionTimeMs,
		FailureReason:   req.FailureReason,
		ErrorDetails:    req.ErrorDetails,
		ContextMetadata: req.ContextMetadata,
	})

	// Upsert rating with confidence score
	var ratingID interface{}
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO ratings (skill_id, revision_id, token_id, score, outcome, task_type, model_used, tokens_consumed, failure_reason, confidence_score, execution_time_ms, error_details, context_metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 ON CONFLICT (revision_id, token_id) DO UPDATE SET
		   score = EXCLUDED.score, outcome = EXCLUDED.outcome,
		   task_type = EXCLUDED.task_type, model_used = EXCLUDED.model_used,
		   tokens_consumed = EXCLUDED.tokens_consumed, failure_reason = EXCLUDED.failure_reason,
		   confidence_score = EXCLUDED.confidence_score, execution_time_ms = EXCLUDED.execution_time_ms,
		   error_details = EXCLUDED.error_details, context_metadata = EXCLUDED.context_metadata,
		   updated_at = NOW()
		 RETURNING id`,
		skillID, revisionID, tokenID, req.Score, req.Outcome,
		req.TaskType, req.ModelUsed, req.TokensConsumed, req.FailureReason,
		confidence, req.ExecutionTimeMs, req.ErrorDetails, req.ContextMetadata).Scan(&ratingID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to submit rating", "")
		return
	}

	// Update rating pattern for anomaly detection
	suspiciousFlags := h.analyzer.CheckTokenHistory(r.Context(), tokenID, skillID)
	go h.analyzer.UpdateRatingPattern(r.Context(), tokenID, skillID, req.Score, suspiciousFlags)

	// Recalculate skill rating and credibility
	go h.refreshSkillRating(skillID, revisionID)

	helpers.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":               ratingID,
		"status":           "recorded",
		"confidence_score": confidence,
		"upsert":           true,
	})
}

func (h *RatingHandler) refreshSkillRating(skillID, revisionID interface{}) {
	ctx := context.Background()
	var count int
	var avgScore, successRate, avgConfidence float64

	// Only count ratings from registered (non-anonymous) tokens
	// Weight ratings by confidence score
	h.pool.QueryRow(ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(r.score * r.confidence_score) / NULLIF(SUM(r.confidence_score), 0), 0),
		        COALESCE(SUM(CASE WHEN r.outcome = 'success' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0),
		        COALESCE(AVG(r.confidence_score), 0.5)
		 FROM ratings r JOIN tokens t ON r.token_id = t.id
		 WHERE r.revision_id = $1 AND t.namespace_id IS NOT NULL`,
		revisionID).Scan(&count, &avgScore, &successRate, &avgConfidence)

	// Bayesian average: (C*m + sum) / (C + n), C=5, m=6.0
	bayesian := 0.0
	if count > 0 {
		bayesian = (5.0*6.0 + avgScore*float64(count)) / (5.0 + float64(count))
	}

	// Calculate skill credibility
	credibility := h.analyzer.CalculateSkillCredibility(ctx, skillID.(uuid.UUID))

	h.pool.Exec(ctx,
		`UPDATE skills SET avg_rating = $2, rating_count = $3, outcome_success_rate = $4, credibility_score = $5, updated_at = NOW() WHERE id = $1`,
		skillID, bayesian, count, successRate, credibility)
}
