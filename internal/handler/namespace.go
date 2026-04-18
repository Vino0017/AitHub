package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type NamespaceHandler struct {
	pool *pgxpool.Pool
}

func NewNamespaceHandler(pool *pgxpool.Pool) *NamespaceHandler {
	return &NamespaceHandler{pool: pool}
}

// Create creates an org namespace. POST /v1/namespaces
func (h *NamespaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON", "")
		return
	}
	if req.Type != "org" {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_type", "Only 'org' type can be created via this endpoint", "")
		return
	}

	callerNsID := middleware.GetNamespaceID(r.Context())

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Transaction failed", "")
		return
	}
	defer tx.Rollback(r.Context())

	var orgID uuid.UUID
	err = tx.QueryRow(r.Context(),
		`INSERT INTO namespaces (name, type) VALUES ($1, 'org') RETURNING id`, req.Name).Scan(&orgID)
	if err != nil {
		helpers.WriteError(w, http.StatusConflict, "name_taken", "Namespace name already taken", "")
		return
	}

	// Add creator as owner
	_, err = tx.Exec(r.Context(),
		`INSERT INTO org_members (org_id, member_id, role) VALUES ($1, $2, 'owner')`, orgID, callerNsID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to add owner", "")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Commit failed", "")
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id": orgID, "name": req.Name, "type": "org",
	})
}

// Get returns namespace info + skills. GET /v1/namespaces/{name}
func (h *NamespaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var id uuid.UUID
	var nsType string
	var createdAt interface{}
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, type, created_at FROM namespaces WHERE name = $1`, name).Scan(&id, &nsType, &createdAt)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "not_found", "Namespace not found", "")
		return
	}

	// List skills (visibility filtered)
	callerNsID := middleware.GetNamespaceID(r.Context())
	rows, err := h.pool.Query(r.Context(),
		`SELECT s.name, s.install_count, s.avg_rating, s.visibility, s.status
		 FROM skills s WHERE s.namespace_id = $1 AND s.status = 'active'
		 AND (s.visibility = 'public' OR s.namespace_id = $2
		      OR EXISTS(SELECT 1 FROM org_members om WHERE om.org_id = s.namespace_id AND om.member_id = $2))
		 ORDER BY s.updated_at DESC`, id, callerNsID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to list skills", "")
		return
	}
	defer rows.Close()

	skills := []map[string]interface{}{}
	for rows.Next() {
		var sName, sVis, sStat string
		var sInstalls int
		var sRating float64
		rows.Scan(&sName, &sInstalls, &sRating, &sVis, &sStat)
		skills = append(skills, map[string]interface{}{
			"name": sName, "installs": sInstalls, "rating": sRating, "visibility": sVis,
		})
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"namespace": name, "type": nsType, "created_at": createdAt, "skills": skills,
	})
}

// AddMember adds a member to an org. POST /v1/namespaces/{name}/members
func (h *NamespaceHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	orgName := chi.URLParam(r, "name")
	var req struct {
		Namespace string `json:"namespace"`
		Role      string `json:"role"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON", "")
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}

	// Verify caller is owner
	var orgID uuid.UUID
	err := h.pool.QueryRow(r.Context(),
		`SELECT id FROM namespaces WHERE name = $1 AND type = 'org'`, orgName).Scan(&orgID)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "org_not_found", "Organization not found", "")
		return
	}

	callerNsID := middleware.GetNamespaceID(r.Context())
	var callerRole string
	err = h.pool.QueryRow(r.Context(),
		`SELECT role FROM org_members WHERE org_id = $1 AND member_id = $2`, orgID, callerNsID).Scan(&callerRole)
	if err != nil || callerRole != "owner" {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "Only org owners can add members", "")
		return
	}

	// Find target namespace
	var memberID uuid.UUID
	err = h.pool.QueryRow(r.Context(),
		`SELECT id FROM namespaces WHERE name = $1 AND type = 'personal'`, req.Namespace).Scan(&memberID)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "member_not_found", "Namespace not found", "")
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`INSERT INTO org_members (org_id, member_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (org_id, member_id) DO UPDATE SET role = $3`, orgID, memberID, req.Role)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to add member", "")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// RemoveMember removes a member. DELETE /v1/namespaces/{name}/members/{memberId}
func (h *NamespaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	orgName := chi.URLParam(r, "name")
	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid member ID", "")
		return
	}

	var orgID uuid.UUID
	err = h.pool.QueryRow(r.Context(),
		`SELECT id FROM namespaces WHERE name = $1 AND type = 'org'`, orgName).Scan(&orgID)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "org_not_found", "Organization not found", "")
		return
	}

	// Verify caller is owner
	callerNsID := middleware.GetNamespaceID(r.Context())
	var callerRole string
	err = h.pool.QueryRow(r.Context(),
		`SELECT role FROM org_members WHERE org_id = $1 AND member_id = $2`, orgID, callerNsID).Scan(&callerRole)
	if err != nil || callerRole != "owner" {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "Only org owners can remove members", "")
		return
	}

	// Check orphan org protection - can't remove last owner
	var memberRole string
	h.pool.QueryRow(r.Context(),
		`SELECT role FROM org_members WHERE org_id = $1 AND member_id = $2`, orgID, memberID).Scan(&memberRole)
	if memberRole == "owner" {
		var ownerCount int
		h.pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND role = 'owner'`, orgID).Scan(&ownerCount)
		if ownerCount <= 1 {
			helpers.WriteError(w, http.StatusForbidden, "last_owner",
				"Cannot remove the last owner. Transfer ownership first or dissolve the org.", "")
			return
		}
	}

	h.pool.Exec(r.Context(), `DELETE FROM org_members WHERE org_id = $1 AND member_id = $2`, orgID, memberID)
	w.WriteHeader(http.StatusNoContent)
}
