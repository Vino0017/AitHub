package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type ForkHandler struct {
	pool *pgxpool.Pool
}

func NewForkHandler(pool *pgxpool.Pool) *ForkHandler {
	return &ForkHandler{pool: pool}
}

// Fork creates a fork of a skill. POST /v1/skills/{namespace}/{name}/fork
func (h *ForkHandler) Fork(w http.ResponseWriter, r *http.Request) {
	nsName := chi.URLParam(r, "namespace")
	skillName := chi.URLParam(r, "name")

	callerNsID := middleware.GetNamespaceID(r.Context())
	callerNsName := middleware.GetNamespaceName(r.Context())
	tokenID := middleware.GetTokenID(r.Context())

	// Find original skill
	var origID, origNsID uuid.UUID
	var desc, fw, vis string
	var tags []string
	err := h.pool.QueryRow(r.Context(),
		`SELECT s.id, s.namespace_id, s.description, s.framework, s.tags, s.visibility
		 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE n.name = $1 AND s.name = $2 AND s.status = 'active'`,
		nsName, skillName).Scan(&origID, &origNsID, &desc, &fw, &tags, &vis)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "skill_not_found", "Skill not found", "")
		return
	}

	// Enforce visibility: only public skills can be forked freely
	if vis == "private" && callerNsID != origNsID {
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "Cannot fork a private skill", "")
		return
	}
	if vis == "org" {
		var isMember bool
		h.pool.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM org_members WHERE org_id = $1 AND member_id = $2)`,
			origNsID, callerNsID).Scan(&isMember)
		if !isMember {
			helpers.WriteError(w, http.StatusForbidden, "forbidden", "Not a member of this organization", "")
			return
		}
	}

	// Get latest approved revision content
	var content, version string
	err = h.pool.QueryRow(r.Context(),
		`SELECT content, version FROM revisions WHERE skill_id = $1 AND review_status = 'approved'
		 ORDER BY created_at DESC LIMIT 1`, origID).Scan(&content, &version)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "no_content", "No approved revision to fork", "")
		return
	}

	// Create forked skill
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Transaction failed", "")
		return
	}
	defer tx.Rollback(r.Context())

	var newSkillID uuid.UUID
	err = tx.QueryRow(r.Context(),
		`INSERT INTO skills (namespace_id, name, description, tags, framework, visibility, forked_from, latest_version)
		 VALUES ($1, $2, $3, $4, $5, 'public', $6, $7) RETURNING id`,
		callerNsID, skillName, desc, tags, fw, origID, "1.0.0").Scan(&newSkillID)
	if err != nil {
		helpers.WriteError(w, http.StatusConflict, "skill_exists",
			"You already have a skill named "+skillName+". Fork creates "+callerNsName+"/"+skillName, "")
		return
	}

	// Create initial revision
	var revID uuid.UUID
	err = tx.QueryRow(r.Context(),
		`INSERT INTO revisions (skill_id, version, content, change_summary, author_token_id, review_status, schema_type)
		 VALUES ($1, '1.0.0', $2, $3, $4, 'approved', 'skill-md') RETURNING id`,
		newSkillID, content, "Forked from "+nsName+"/"+skillName+" v"+version, tokenID).Scan(&revID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to create revision", "")
		return
	}

	// Increment original fork count
	tx.Exec(r.Context(), `UPDATE skills SET fork_count = fork_count + 1 WHERE id = $1`, origID)

	if err := tx.Commit(r.Context()); err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Commit failed", "")
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          newSkillID.String(),
		"full_name":   callerNsName + "/" + skillName,
		"forked_from": nsName + "/" + skillName,
		"version":     "1.0.0",
	})
}

// ListForks lists forks of a skill. GET /v1/skills/{namespace}/{name}/forks
func (h *ForkHandler) ListForks(w http.ResponseWriter, r *http.Request) {
	skillID, ok := checkSkillAccess(h.pool, w, r)
	if !ok {
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT s.id, n.name, s.name, s.avg_rating, s.install_count, s.created_at
		 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE s.forked_from = $1 AND s.status = 'active'
		 ORDER BY s.install_count DESC`, skillID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to list forks", "")
		return
	}
	defer rows.Close()

	forks := []map[string]interface{}{}
	for rows.Next() {
		var id uuid.UUID
		var ns, name string
		var rating float64
		var installs int
		var createdAt interface{}
		rows.Scan(&id, &ns, &name, &rating, &installs, &createdAt)
		forks = append(forks, map[string]interface{}{
			"id": id, "full_name": ns + "/" + name, "avg_rating": rating,
			"install_count": installs, "created_at": createdAt,
		})
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{"forks": forks})
}

// GetForkTree returns the complete fork tree (ancestors and descendants).
// GET /v1/skills/{namespace}/{name}/fork-tree
func (h *ForkHandler) GetForkTree(w http.ResponseWriter, r *http.Request) {
	skillID, ok := checkSkillAccess(h.pool, w, r)
	if !ok {
		return
	}

	// Get ancestors (parent, grandparent, etc.)
	ancestors := h.getAncestors(r.Context(), skillID)

	// Get descendants (direct forks and their forks)
	descendants := h.getDescendants(r.Context(), skillID, 0, 3) // Max depth 3

	// Get current skill info
	var ns, name string
	var rating float64
	var installs, forkCount int
	h.pool.QueryRow(r.Context(),
		`SELECT n.name, s.name, s.avg_rating, s.install_count, s.fork_count
		 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE s.id = $1`, skillID).Scan(&ns, &name, &rating, &installs, &forkCount)

	current := map[string]interface{}{
		"id":           skillID,
		"full_name":    ns + "/" + name,
		"avg_rating":   rating,
		"install_count": installs,
		"fork_count":   forkCount,
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"current":     current,
		"ancestors":   ancestors,
		"descendants": descendants,
	})
}

func (h *ForkHandler) getAncestors(ctx context.Context, skillID uuid.UUID) []map[string]interface{} {
	ancestors := []map[string]interface{}{}

	currentID := skillID
	for i := 0; i < 10; i++ { // Max 10 levels up
		var parentID *uuid.UUID
		var ns, name string
		var rating float64
		var installs int

		err := h.pool.QueryRow(ctx,
			`SELECT s.forked_from, n.name, s.name, s.avg_rating, s.install_count
			 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
			 WHERE s.id = $1`, currentID).Scan(&parentID, &ns, &name, &rating, &installs)
		if err != nil || parentID == nil {
			break
		}

		ancestors = append([]map[string]interface{}{{
			"id":           *parentID,
			"full_name":    ns + "/" + name,
			"avg_rating":   rating,
			"install_count": installs,
		}}, ancestors...) // Prepend to get root first

		currentID = *parentID
	}

	return ancestors
}

func (h *ForkHandler) getDescendants(ctx context.Context, skillID uuid.UUID, depth, maxDepth int) []map[string]interface{} {
	if depth >= maxDepth {
		return []map[string]interface{}{}
	}

	rows, err := h.pool.Query(ctx,
		`SELECT s.id, n.name, s.name, s.avg_rating, s.install_count, s.fork_count
		 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE s.forked_from = $1 AND s.status = 'active'
		 ORDER BY s.install_count DESC`, skillID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()

	descendants := []map[string]interface{}{}
	for rows.Next() {
		var id uuid.UUID
		var ns, name string
		var rating float64
		var installs, forkCount int
		rows.Scan(&id, &ns, &name, &rating, &installs, &forkCount)

		fork := map[string]interface{}{
			"id":           id,
			"full_name":    ns + "/" + name,
			"avg_rating":   rating,
			"install_count": installs,
			"fork_count":   forkCount,
			"depth":        depth + 1,
		}

		// Recursively get children
		children := h.getDescendants(ctx, id, depth+1, maxDepth)
		if len(children) > 0 {
			fork["children"] = children
		}

		descendants = append(descendants, fork)
	}

	return descendants
}

// GetForkRanking returns forks ranked by quality (rating * success_rate * installs).
// GET /v1/skills/{namespace}/{name}/fork-ranking
func (h *ForkHandler) GetForkRanking(w http.ResponseWriter, r *http.Request) {
	skillID, ok := checkSkillAccess(h.pool, w, r)
	if !ok {
		return
	}

	// Get all forks in the tree (including transitive forks)
	allForks := h.getAllForksInTree(r.Context(), skillID)

	// Rank by quality score
	rows, err := h.pool.Query(r.Context(),
		`SELECT s.id, n.name, s.name, s.avg_rating, s.outcome_success_rate, s.install_count, s.fork_count, s.created_at
		 FROM skills s JOIN namespaces n ON s.namespace_id = n.id
		 WHERE s.id = ANY($1) AND s.status = 'active'
		 ORDER BY (s.avg_rating * s.outcome_success_rate * LOG(1 + s.install_count)) DESC
		 LIMIT 20`, allForks)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to rank forks", "")
		return
	}
	defer rows.Close()

	ranking := []map[string]interface{}{}
	rank := 1
	for rows.Next() {
		var id uuid.UUID
		var ns, name string
		var rating, successRate float64
		var installs, forkCount int
		var createdAt interface{}
		rows.Scan(&id, &ns, &name, &rating, &successRate, &installs, &forkCount, &createdAt)

		ranking = append(ranking, map[string]interface{}{
			"rank":                rank,
			"id":                  id,
			"full_name":           ns + "/" + name,
			"avg_rating":          rating,
			"outcome_success_rate": successRate,
			"install_count":       installs,
			"fork_count":          forkCount,
			"quality_score":       rating * successRate * (1 + float64(installs)),
			"created_at":          createdAt,
		})
		rank++
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{"ranking": ranking})
}

func (h *ForkHandler) getAllForksInTree(ctx context.Context, rootID uuid.UUID) []uuid.UUID {
	// BFS to collect all forks
	visited := make(map[uuid.UUID]bool)
	queue := []uuid.UUID{rootID}
	allForks := []uuid.UUID{}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true
		allForks = append(allForks, current)

		// Get direct forks
		rows, err := h.pool.Query(ctx,
			`SELECT id FROM skills WHERE forked_from = $1 AND status = 'active'`, current)
		if err != nil {
			continue
		}

		for rows.Next() {
			var forkID uuid.UUID
			rows.Scan(&forkID)
			if !visited[forkID] {
				queue = append(queue, forkID)
			}
		}
		rows.Close()
	}

	return allForks
}
