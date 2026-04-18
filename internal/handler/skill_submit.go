package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
	"github.com/skillhub/api/internal/review"
	"github.com/skillhub/api/internal/skillformat"
)

type SkillSubmitHandler struct {
	pool     *pgxpool.Pool
	reviewer *review.Reviewer
	river    *river.Client[pgx.Tx]
}

func NewSkillSubmitHandler(pool *pgxpool.Pool, reviewer *review.Reviewer, riverClient *river.Client[pgx.Tx]) *SkillSubmitHandler {
	return &SkillSubmitHandler{pool: pool, reviewer: reviewer, river: riverClient}
}

// Submit handles POST /v1/skills
func (h *SkillSubmitHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content    string `json:"content"`
		Visibility string `json:"visibility,omitempty"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body", "")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		helpers.WriteError(w, http.StatusBadRequest, "empty_content", "Content cannot be empty", "")
		return
	}

	// Parse and validate frontmatter
	fm, _, err := skillformat.Parse(req.Content)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_format", err.Error(), "")
		return
	}

	// Validate required fields
	if err := skillformat.Validate(fm); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error(), "")
		return
	}

	nsID := middleware.GetNamespaceID(r.Context())
	tokenID := middleware.GetTokenID(r.Context())

	visibility := req.Visibility
	if visibility == "" {
		visibility = "public"
	}
	if visibility != "public" && visibility != "private" && visibility != "org" {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_visibility", "Visibility must be public, private, or org", "")
		return
	}

	// Check if skill already exists in this namespace
	var existingID *uuid.UUID
	h.pool.QueryRow(r.Context(),
		`SELECT id FROM skills WHERE namespace_id = $1 AND name = $2`, nsID, fm.Name).Scan(&existingID)

	schema := fm.Schema
	if schema == "" {
		schema = "skill-md"
	}

	reqJSON, _ := json.Marshal(fm.Requirements)
	var platformJSON []byte
	if fm.Requirements != nil && fm.Requirements.Platform != nil {
		platformJSON, _ = json.Marshal(fm.Requirements.Platform)
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to start transaction", "")
		return
	}
	defer tx.Rollback(r.Context())

	var skillID uuid.UUID
	if existingID != nil {
		// Skill exists, create new revision — enforce version > latest
		skillID = *existingID

		var latestVersion string
		h.pool.QueryRow(r.Context(),
			`SELECT latest_version FROM skills WHERE id = $1`, skillID).Scan(&latestVersion)

		if latestVersion != "" {
			cmp, err := skillformat.CompareSemVer(fm.Version, latestVersion)
			if err == nil && cmp <= 0 {
				helpers.WriteError(w, http.StatusBadRequest, "version_too_low",
					"Version "+fm.Version+" must be greater than current latest "+latestVersion, "")
				return
			}
		}
	} else {
		// Create new skill
		err = tx.QueryRow(r.Context(),
			`INSERT INTO skills (namespace_id, name, description, tags, framework, visibility, latest_version)
			 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			nsID, fm.Name, fm.Description, fm.Tags, fm.Framework, visibility, fm.Version).Scan(&skillID)
		if err != nil {
			helpers.WriteError(w, http.StatusConflict, "skill_exists", "Skill name already taken in this namespace", "")
			return
		}
	}

	// Create revision
	triggers := fm.Triggers
	if triggers == nil {
		triggers = []string{}
	}
	models := fm.CompatibleModels
	if models == nil {
		models = []string{}
	}

	var revID uuid.UUID
	err = tx.QueryRow(r.Context(),
		`INSERT INTO revisions (skill_id, version, content, change_summary, author_token_id, review_status,
		 schema_type, triggers, compatible_models, estimated_tokens, requirements, platform)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8, $9, $10, $11) RETURNING id`,
		skillID, fm.Version, req.Content, "Initial submission",
		tokenID, schema, triggers, models, fm.EstimatedTokens,
		reqJSON, platformJSON).Scan(&revID)
	if err != nil {
		if strings.Contains(err.Error(), "revisions_skill_version_unique") {
			helpers.WriteError(w, http.StatusConflict, "version_exists",
				"Version "+fm.Version+" already exists. Please increment the version number.", "")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to create revision: "+err.Error(), "")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to commit", "")
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
		"id":       skillID.String(),
		"revision": revID.String(),
		"status":   "pending",
		"version":  fm.Version,
	})
}
