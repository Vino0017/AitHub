package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
	"github.com/skillhub/api/internal/review"
	"github.com/skillhub/api/internal/skillformat"
)

type RevisionHandler struct {
	pool     *pgxpool.Pool
	reviewer *review.Reviewer
	river    *river.Client[pgx.Tx]
}

func NewRevisionHandler(pool *pgxpool.Pool, reviewer *review.Reviewer, riverClient *river.Client[pgx.Tx]) *RevisionHandler {
	return &RevisionHandler{pool: pool, reviewer: reviewer, river: riverClient}
}

// List returns revision history. GET /v1/skills/{namespace}/{name}/revisions
func (h *RevisionHandler) List(w http.ResponseWriter, r *http.Request) {
	skillID, ok := checkSkillAccess(h.pool, w, r)
	if !ok {
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, version, change_summary, review_status, created_at
		 FROM revisions WHERE skill_id = $1 ORDER BY created_at DESC`, skillID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to list revisions", "")
		return
	}
	defer rows.Close()

	revisions := []map[string]interface{}{}
	for rows.Next() {
		var id uuid.UUID
		var version, summary, status string
		var createdAt interface{}
		rows.Scan(&id, &version, &summary, &status, &createdAt)
		revisions = append(revisions, map[string]interface{}{
			"id": id, "version": version, "change_summary": summary,
			"review_status": status, "created_at": createdAt,
		})
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{"revisions": revisions})
}

// GetVersion returns a specific revision. GET /v1/skills/{namespace}/{name}/revisions/{version}
func (h *RevisionHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")

	skillID, ok := checkSkillAccess(h.pool, w, r)
	if !ok {
		return
	}

	var id uuid.UUID
	var content, summary, status, schemaType string
	var triggers, models []string
	var tokens int
	var reqs, platform, feedback interface{}
	var createdAt interface{}

	err := h.pool.QueryRow(r.Context(),
		`SELECT id, content, change_summary, review_status, schema_type, triggers,
		        compatible_models, estimated_tokens, requirements, platform,
		        review_feedback, created_at
		 FROM revisions WHERE skill_id = $1 AND version = $2`, skillID, version).Scan(
		&id, &content, &summary, &status, &schemaType, &triggers, &models,
		&tokens, &reqs, &platform, &feedback, &createdAt)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "revision_not_found", "Version not found", "")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"id": id, "version": version, "content": content, "change_summary": summary,
		"review_status": status, "schema_type": schemaType, "triggers": triggers,
		"compatible_models": models, "estimated_tokens": tokens, "requirements": reqs,
		"platform": platform, "review_feedback": feedback, "created_at": createdAt,
	})
}

// Submit creates a new revision. POST /v1/skills/{namespace}/{name}/revisions
func (h *RevisionHandler) Submit(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")

	// Verify ownership
	callerNs := middleware.GetNamespaceName(r.Context())
	if callerNs != nsName {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "You can only submit revisions to your own skills", "")
		return
	}

	var req struct {
		Content       string `json:"content"`
		ChangeSummary string `json:"change_summary"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body", "")
		return
	}

	fm, _, err := skillformat.Parse(req.Content)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_format", err.Error(), "")
		return
	}
	if err := skillformat.Validate(fm); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error(), "")
		return
	}

	var skillID uuid.UUID
	var latestVersion string
	err = h.pool.QueryRow(r.Context(),
		`SELECT s.id, s.latest_version FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE n.name = $1 AND s.name = $2`, nsName, skillName).Scan(&skillID, &latestVersion)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "skill_not_found", "Skill not found", "")
		return
	}

	// Enforce version > latest_version (prevent rollback)
	if latestVersion != "" {
		cmp, err := skillformat.CompareSemVer(fm.Version, latestVersion)
		if err == nil && cmp <= 0 {
			helpers.WriteError(w, http.StatusBadRequest, "version_too_low",
				"Version "+fm.Version+" must be greater than current latest "+latestVersion, "")
			return
		}
	}

	tokenID := middleware.GetTokenID(r.Context())
	schema := fm.Schema
	if schema == "" {
		schema = "skill-md"
	}

	changeSummary := req.ChangeSummary
	if changeSummary == "" {
		changeSummary = "Updated to " + fm.Version
	}

	reqJSON, _ := json.Marshal(fm.Requirements)
	var platformJSON []byte
	if fm.Requirements != nil && fm.Requirements.Platform != nil {
		platformJSON, _ = json.Marshal(fm.Requirements.Platform)
	}

	triggers := fm.Triggers
	if triggers == nil {
		triggers = []string{}
	}
	compatModels := fm.CompatibleModels
	if compatModels == nil {
		compatModels = []string{}
	}

	var revID uuid.UUID
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO revisions (skill_id, version, content, change_summary, author_token_id, review_status,
		 schema_type, triggers, compatible_models, estimated_tokens, requirements, platform)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8, $9, $10, $11) RETURNING id`,
		skillID, fm.Version, req.Content, changeSummary, tokenID, schema,
		triggers, compatModels, fm.EstimatedTokens, reqJSON, platformJSON).Scan(&revID)
	if err != nil {
		if strings.Contains(err.Error(), "revisions_skill_version_unique") {
			helpers.WriteError(w, http.StatusConflict, "version_exists",
				"Version "+fm.Version+" already exists. Please use a new version number.", "")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to create revision", "")
		return
	}

	// Enqueue review job via River (persistent, crash-safe)
	_, err = h.river.Insert(context.Background(), review.ReviewJobArgs{
		RevisionID: revID.String(),
	}, nil)
	if err != nil {
		log.Printf("warning: failed to enqueue review job: %v, falling back to goroutine", err)
		go h.reviewer.Review(context.Background(), revID)
	}

	helpers.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"revision": revID.String(), "version": fm.Version, "status": "pending",
	})
}
