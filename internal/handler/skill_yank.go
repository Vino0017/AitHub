package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type SkillYankHandler struct {
	pool *pgxpool.Pool
}

func NewSkillYankHandler(pool *pgxpool.Pool) *SkillYankHandler {
	return &SkillYankHandler{pool: pool}
}

// Yank sets a skill to yanked status. DELETE /v1/skills/{namespace}/{name}
func (h *SkillYankHandler) Yank(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")

	callerNs := middleware.GetNamespaceName(r.Context())
	if callerNs != nsName {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "You can only yank your own skills", "")
		return
	}

	tag, err := h.pool.Exec(r.Context(),
		`UPDATE skills SET status = 'yanked', updated_at = NOW()
		 FROM namespaces n WHERE skills.namespace_id = n.id AND n.name = $1 AND skills.name = $2
		 AND skills.status = 'active'`, nsName, skillName)
	if err != nil || tag.RowsAffected() == 0 {
		helpers.WriteError(w, http.StatusNotFound, "skill_not_found", "Skill not found or already yanked", "")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "yanked"})
}

// Restore restores a yanked skill. PATCH /v1/skills/{namespace}/{name}
func (h *SkillYankHandler) Restore(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")

	callerNs := middleware.GetNamespaceName(r.Context())
	if callerNs != nsName {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "You can only restore your own skills", "")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil || req.Status != "active" {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_status", "Only status 'active' is supported for restore", "")
		return
	}

	tag, err := h.pool.Exec(r.Context(),
		`UPDATE skills SET status = 'active', updated_at = NOW()
		 FROM namespaces n WHERE skills.namespace_id = n.id AND n.name = $1 AND skills.name = $2
		 AND skills.status = 'yanked'`, nsName, skillName)
	if err != nil || tag.RowsAffected() == 0 {
		helpers.WriteError(w, http.StatusNotFound, "not_found", "Skill not found or not in yanked state", "")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "active"})
}
