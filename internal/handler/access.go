package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

// checkSkillAccess verifies the caller has visibility access to the skill.
// Returns the skill ID on success, or writes an error response and returns uuid.Nil on failure.
func checkSkillAccess(pool *pgxpool.Pool, w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")

	var skillID, nsID uuid.UUID
	var vis, status string
	err := pool.QueryRow(r.Context(),
		`SELECT s.id, s.namespace_id, s.visibility, s.status
		 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE n.name = $1 AND s.name = $2`, nsName, skillName).Scan(&skillID, &nsID, &vis, &status)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "skill_not_found", "Skill not found", "")
		return uuid.Nil, false
	}

	if status != "active" {
		helpers.WriteError(w, http.StatusGone, "skill_unavailable", "This skill has been removed or yanked", "")
		return uuid.Nil, false
	}

	callerNsID := middleware.GetNamespaceID(r.Context())

	if vis == "private" && callerNsID != nsID {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "This skill is private", "")
		return uuid.Nil, false
	}
	if vis == "org" {
		var isMember bool
		pool.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM org_members WHERE org_id = $1 AND member_id = $2)`,
			nsID, callerNsID).Scan(&isMember)
		if !isMember {
			helpers.WriteError(w, http.StatusForbidden, "forbidden", "Not a member of this organization", "")
			return uuid.Nil, false
		}
	}

	return skillID, true
}
