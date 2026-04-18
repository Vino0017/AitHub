package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type SkillDetailHandler struct {
	pool *pgxpool.Pool
}

func NewSkillDetailHandler(pool *pgxpool.Pool) *SkillDetailHandler {
	return &SkillDetailHandler{pool: pool}
}

// Get returns skill details. GET /v1/skills/{namespace}/{name}
func (h *SkillDetailHandler) Get(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")

	skill, err := h.getSkillWithAccess(w, r, nsName, skillName)
	if err != nil {
		return
	}

	var revRequirements, revPlatform interface{}
	var revVersion, revSchemaType string
	var revTriggers, revModels []string
	var revTokens int

	err2 := h.pool.QueryRow(r.Context(),
		`SELECT version, schema_type, triggers, compatible_models, estimated_tokens, requirements, platform
		 FROM revisions WHERE skill_id = $1 AND review_status = 'approved'
		 ORDER BY created_at DESC LIMIT 1`, skill["id"]).Scan(
		&revVersion, &revSchemaType, &revTriggers, &revModels, &revTokens, &revRequirements, &revPlatform)
	if err2 == nil {
		skill["schema_type"] = revSchemaType
		skill["triggers"] = revTriggers
		skill["compatible_models"] = revModels
		skill["estimated_tokens"] = revTokens
		skill["requirements"] = revRequirements
		skill["platform"] = revPlatform
	}

	var revCount int
	h.pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM revisions WHERE skill_id = $1`, skill["id"]).Scan(&revCount)
	skill["revision_count"] = revCount

	helpers.WriteJSON(w, http.StatusOK, skill)
}

// Content returns SKILL.md content, increments install_count. GET /v1/skills/{namespace}/{name}/content?version=1.0.0
func (h *SkillDetailHandler) Content(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")
	requestedVersion := r.URL.Query().Get("version")

	skill, err := h.getSkillWithAccess(w, r, nsName, skillName)
	if err != nil {
		return
	}

	var content, version string
	var query string
	var args []interface{}

	if requestedVersion != "" {
		// Version-locked request
		query = `SELECT content, version FROM revisions WHERE skill_id = $1 AND version = $2 AND review_status = 'approved'`
		args = []interface{}{skill["id"], requestedVersion}
	} else {
		// Latest version
		query = `SELECT content, version FROM revisions WHERE skill_id = $1 AND review_status = 'approved' ORDER BY created_at DESC LIMIT 1`
		args = []interface{}{skill["id"]}
	}

	err2 := h.pool.QueryRow(r.Context(), query, args...).Scan(&content, &version)
	if err2 != nil {
		if requestedVersion != "" {
			helpers.WriteError(w, http.StatusNotFound, "version_not_found", "Version not found or not approved", "")
		} else {
			helpers.WriteError(w, http.StatusNotFound, "no_approved_revision", "No approved revision found", "")
		}
		return
	}

	go h.pool.Exec(context.Background(), `UPDATE skills SET install_count = install_count + 1 WHERE id = $1`, skill["id"])

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"namespace": nsName, "name": skillName, "version": version, "content": content,
	})
}

// Status returns latest revision review status. GET /v1/skills/{namespace}/{name}/status
func (h *SkillDetailHandler) Status(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")

	skill, err := h.getSkillWithAccess(w, r, nsName, skillName)
	if err != nil {
		return
	}

	var version, reviewStatus string
	var reviewFeedback interface{}
	err2 := h.pool.QueryRow(r.Context(),
		`SELECT version, review_status, review_feedback FROM revisions WHERE skill_id = $1 ORDER BY created_at DESC LIMIT 1`,
		skill["id"]).Scan(&version, &reviewStatus, &reviewFeedback)
	if err2 != nil {
		helpers.WriteError(w, http.StatusNotFound, "no_revision", "No revision found", "")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": reviewStatus, "version": version, "review_feedback": reviewFeedback,
	})
}

// Updates checks for new versions. GET /v1/skills/{namespace}/{name}/updates?current_version=1.0.0
func (h *SkillDetailHandler) Updates(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")
	currentVersion := r.URL.Query().Get("current_version")

	if currentVersion == "" {
		helpers.WriteError(w, http.StatusBadRequest, "missing_version", "current_version parameter required", "")
		return
	}

	skill, err := h.getSkillWithAccess(w, r, nsName, skillName)
	if err != nil {
		return
	}

	// Get all newer approved revisions
	rows, err := h.pool.Query(r.Context(),
		`SELECT version, breaking_change, migration_guide, change_summary, created_at
		 FROM revisions
		 WHERE skill_id = $1 AND review_status = 'approved'
		 AND created_at > (SELECT created_at FROM revisions WHERE skill_id = $1 AND version = $2)
		 ORDER BY created_at ASC`,
		skill["id"], currentVersion)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to check updates", "")
		return
	}
	defer rows.Close()

	updates := []map[string]interface{}{}
	for rows.Next() {
		var version, changeSummary string
		var migrationGuide interface{}
		var breakingChange bool
		var createdAt interface{}

		rows.Scan(&version, &breakingChange, &migrationGuide, &changeSummary, &createdAt)

		update := map[string]interface{}{
			"version":          version,
			"breaking_change":  breakingChange,
			"change_summary":   changeSummary,
			"created_at":       createdAt,
		}
		if migrationGuide != nil {
			update["migration_guide"] = migrationGuide
		}

		updates = append(updates, update)
	}

	hasUpdates := len(updates) > 0
	var latestVersion string
	if hasUpdates {
		latestVersion = updates[len(updates)-1]["version"].(string)
	} else {
		latestVersion = currentVersion
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"current_version": currentVersion,
		"latest_version":  latestVersion,
		"has_updates":     hasUpdates,
		"updates":         updates,
	})
}

func (h *SkillDetailHandler) getSkillWithAccess(w http.ResponseWriter, r *http.Request, nsName, skillName string) (map[string]interface{}, error) {
	var id, nsID, forkedFrom interface{}
	var name, desc, fw, vis, lv, status string
	var tags []string
	var installs, ratingCount, forkCount int
	var avgRating, successRate float64
	var createdAt, updatedAt interface{}

	err := h.pool.QueryRow(r.Context(),
		`SELECT s.id, s.namespace_id, s.name, s.description, s.tags, s.framework,
		        s.visibility, s.forked_from, s.install_count, s.avg_rating,
		        s.rating_count, s.outcome_success_rate, s.latest_version,
		        s.fork_count, s.status, s.created_at, s.updated_at
		 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE n.name = $1 AND s.name = $2`, nsName, skillName).Scan(
		&id, &nsID, &name, &desc, &tags, &fw, &vis, &forkedFrom,
		&installs, &avgRating, &ratingCount, &successRate, &lv,
		&forkCount, &status, &createdAt, &updatedAt)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "skill_not_found", "Skill not found", "")
		return nil, err
	}

	if status != "active" {
		helpers.WriteError(w, http.StatusGone, "skill_unavailable", "This skill has been removed or yanked", "")
		return nil, fmt.Errorf("not active")
	}

	callerNsID := middleware.GetNamespaceID(r.Context())
	if vis == "private" && callerNsID != nsID {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "This skill is private", "")
		return nil, fmt.Errorf("forbidden")
	}
	if vis == "org" {
		var isMember bool
		h.pool.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM org_members WHERE org_id = $1 AND member_id = $2)`,
			nsID, callerNsID).Scan(&isMember)
		if !isMember {
			helpers.WriteError(w, http.StatusForbidden, "forbidden", "Not a member of this organization", "")
			return nil, fmt.Errorf("forbidden")
		}
	}

	return map[string]interface{}{
		"id": id, "namespace": nsName, "name": name, "full_name": nsName + "/" + name,
		"description": desc, "tags": tags, "framework": fw, "visibility": vis,
		"forked_from": forkedFrom, "install_count": installs, "avg_rating": avgRating,
		"rating_count": ratingCount, "outcome_success_rate": successRate,
		"latest_version": lv, "fork_count": forkCount, "status": status,
		"created_at": createdAt, "updated_at": updatedAt,
	}, nil
}
