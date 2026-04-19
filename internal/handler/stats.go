package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
)

type StatsHandler struct {
	pool *pgxpool.Pool
}

func NewStatsHandler(pool *pgxpool.Pool) *StatsHandler {
	return &StatsHandler{pool: pool}
}

// GetGlobalStats returns global platform statistics
func (h *StatsHandler) GetGlobalStats(w http.ResponseWriter, r *http.Request) {
	var (
		totalSkills       int
		totalInstalls     int64
		totalContributors int
	)

	// Get total active skills
	err := h.pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM skills WHERE status = 'active'
	`).Scan(&totalSkills)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to fetch skills count", "")
		return
	}

	// Get total installs across all skills
	err = h.pool.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(install_count), 0) FROM skills WHERE status = 'active'
	`).Scan(&totalInstalls)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to fetch install count", "")
		return
	}

	// Get total unique contributors (namespace owners)
	err = h.pool.QueryRow(r.Context(), `
		SELECT COUNT(DISTINCT namespace_id) FROM skills WHERE status = 'active'
	`).Scan(&totalContributors)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to fetch contributors count", "")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total_skills":       totalSkills,
		"total_installs":     totalInstalls,
		"total_contributors": totalContributors,
	})
}
