package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
)

type AdminHandler struct {
	pool *pgxpool.Pool
}

func NewAdminHandler(pool *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{pool: pool}
}

// ListPending lists skills pending review. GET /admin/skills/pending
func (h *AdminHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT r.id, r.skill_id, r.version, n.name AS ns, s.name AS skill, r.review_status, r.created_at
		 FROM revisions r
		 JOIN skills s ON r.skill_id = s.id
		 JOIN namespaces n ON s.namespace_id = n.id
		 WHERE r.review_status IN ('pending', 'revision_requested')
		 ORDER BY r.created_at ASC LIMIT 50`)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Query failed", "")
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var id, skillID uuid.UUID
		var version, ns, skill, status string
		var createdAt interface{}
		rows.Scan(&id, &skillID, &version, &ns, &skill, &status, &createdAt)
		items = append(items, map[string]interface{}{
			"revision_id": id, "skill": ns + "/" + skill, "version": version,
			"status": status, "created_at": createdAt,
		})
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{"pending": items})
}

// Approve approves a revision. POST /admin/skills/{id}/approve
func (h *AdminHandler) Approve(w http.ResponseWriter, r *http.Request) {
	revID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid revision ID", "")
		return
	}

	var skillID uuid.UUID
	var version string
	err = h.pool.QueryRow(r.Context(),
		`UPDATE revisions SET review_status = 'approved', review_result = '{"approved_by":"admin"}'
		 WHERE id = $1 RETURNING skill_id, version`, revID).Scan(&skillID, &version)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "not_found", "Revision not found", "")
		return
	}

	h.pool.Exec(r.Context(),
		`UPDATE skills SET latest_version = $2, updated_at = NOW() WHERE id = $1`, skillID, version)

	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// Reject rejects a revision. POST /admin/skills/{id}/reject
func (h *AdminHandler) Reject(w http.ResponseWriter, r *http.Request) {
	revID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid revision ID", "")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	helpers.ReadJSON(r, &req)

	h.pool.Exec(r.Context(),
		`UPDATE revisions SET review_status = 'rejected', review_result = $2 WHERE id = $1`,
		revID, `{"reason":"`+req.Reason+`"}`)

	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// RemoveSkill admin-removes a skill. POST /admin/skills/{id}/remove
func (h *AdminHandler) RemoveSkill(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	h.pool.Exec(r.Context(),
		`UPDATE skills SET status = 'removed', updated_at = NOW() WHERE id = $1`, idStr)
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// BanNamespace bans a namespace. POST /admin/namespaces/{id}/ban
func (h *AdminHandler) BanNamespace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	h.pool.Exec(r.Context(), `UPDATE namespaces SET banned = TRUE WHERE id = $1`, idStr)
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "banned"})
}

// RefreshRatings recalculates all skill ratings. POST /admin/ratings/refresh
func (h *AdminHandler) RefreshRatings(w http.ResponseWriter, r *http.Request) {
	go h.refreshAllRatings()
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "refresh_started"})
}

func (h *AdminHandler) refreshAllRatings() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := h.pool.Exec(ctx,
		`UPDATE skills s SET
		    avg_rating = sub.bayesian_avg,
		    rating_count = sub.n,
		    outcome_success_rate = sub.success_rate,
		    updated_at = NOW()
		 FROM (
		    SELECT sk.id AS skill_id,
		           COALESCE(stats.n, 0) AS n,
		           CASE WHEN COALESCE(stats.n, 0) = 0 THEN 0
		                ELSE (5.0 * 6.0 + COALESCE(stats.total_score, 0)) / (5.0 + COALESCE(stats.n, 0))
		           END AS bayesian_avg,
		           COALESCE(stats.success_rate, 0) AS success_rate
		    FROM skills sk
		    LEFT JOIN LATERAL (
		        SELECT rev.id AS rev_id FROM revisions rev
		        WHERE rev.skill_id = sk.id AND rev.review_status = 'approved'
		        ORDER BY rev.created_at DESC LIMIT 1
		    ) latest_rev ON TRUE
		    LEFT JOIN LATERAL (
		        SELECT COUNT(*)::int AS n,
		               SUM(r.score)::float AS total_score,
		               SUM(CASE WHEN r.outcome = 'success' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) AS success_rate
		        FROM ratings r JOIN tokens t ON r.token_id = t.id
		        WHERE r.revision_id = latest_rev.rev_id AND t.namespace_id IS NOT NULL
		    ) stats ON TRUE
		    WHERE sk.status = 'active'
		 ) sub
		 WHERE s.id = sub.skill_id`)
	if err != nil {
		log.Printf("rating refresh error: %v", err)
	}
}
