package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type SkillSearchHandler struct {
	pool *pgxpool.Pool
}

func NewSkillSearchHandler(pool *pgxpool.Pool) *SkillSearchHandler {
	return &SkillSearchHandler{pool: pool}
}

// Search handles GET /v1/skills
func (h *SkillSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	framework := q.Get("framework")
	tag := q.Get("tag")
	sort := q.Get("sort")
	osFilter := q.Get("os")
	archFilter := q.Get("arch")

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	if sort == "" {
		sort = "rating"
	}

	// Build the caller's namespace ID for visibility filtering
	var callerNsID interface{}
	if !middleware.IsAnonymous(r.Context()) {
		callerNsID = middleware.GetNamespaceID(r.Context())
	}

	// Build ORDER BY
	orderBy := "s.avg_rating DESC"
	switch sort {
	case "installs":
		orderBy = "s.install_count DESC"
	case "recent", "new":
		orderBy = "s.created_at DESC"
	case "trending":
		orderBy = "s.install_count DESC"
	}

	// Build WHERE clauses
	sqlQuery := `
		SELECT s.id, s.namespace_id, n.name AS namespace_name, s.name, s.description,
		       s.tags, s.framework, s.visibility, s.forked_from,
		       s.install_count, s.avg_rating, s.rating_count, s.outcome_success_rate,
		       s.latest_version, s.fork_count, s.status, s.created_at, s.updated_at
		FROM skills s
		JOIN namespaces n ON s.namespace_id = n.id
		WHERE s.status = 'active'`

	args := []interface{}{}
	argIdx := 1

	if query != "" {
		sqlQuery += ` AND s.search_vector @@ plainto_tsquery('simple', $` + strconv.Itoa(argIdx) + `)`
		args = append(args, query)
		argIdx++
	}

	if framework != "" {
		sqlQuery += ` AND s.framework = $` + strconv.Itoa(argIdx)
		args = append(args, framework)
		argIdx++
	}

	if tag != "" {
		sqlQuery += ` AND $` + strconv.Itoa(argIdx) + ` = ANY(s.tags)`
		args = append(args, tag)
		argIdx++
	}

	if sort == "new" {
		sqlQuery += ` AND s.created_at > NOW() - INTERVAL '7 days'`
	}

	// Visibility filter
	if callerNsID != nil {
		sqlQuery += ` AND (s.visibility = 'public' OR (s.visibility = 'private' AND s.namespace_id = $` + strconv.Itoa(argIdx) + `)
		              OR (s.visibility = 'org' AND EXISTS (
		                SELECT 1 FROM org_members om WHERE om.org_id = s.namespace_id AND om.member_id = $` + strconv.Itoa(argIdx) + `)))`
		args = append(args, callerNsID)
		argIdx++
	} else {
		sqlQuery += ` AND s.visibility = 'public'`
	}

	sqlQuery += ` ORDER BY ` + orderBy
	sqlQuery += ` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := h.pool.Query(r.Context(), sqlQuery, args...)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Search failed: "+err.Error(), "")
		return
	}
	defer rows.Close()

	skills := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, nsID                                interface{}
			nsName, name, desc, fw, vis, lv, status string
			tags                                    []string
			forkedFrom                              interface{}
			installs, ratingCount, forkCount         int
			avgRating, successRate                   float64
			createdAt, updatedAt                     interface{}
		)
		err := rows.Scan(&id, &nsID, &nsName, &name, &desc, &tags, &fw, &vis, &forkedFrom,
			&installs, &avgRating, &ratingCount, &successRate, &lv, &forkCount, &status, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		skill := map[string]interface{}{
			"id":                   id,
			"namespace":            nsName,
			"name":                 name,
			"full_name":            nsName + "/" + name,
			"description":          desc,
			"tags":                 tags,
			"framework":            fw,
			"visibility":           vis,
			"latest_version":       lv,
			"avg_rating":           avgRating,
			"rating_count":         ratingCount,
			"install_count":        installs,
			"outcome_success_rate": successRate,
			"fork_count":           forkCount,
			"created_at":           createdAt,
		}

		// Platform filtering (post-query for now, can optimize with JSON query later)
		if osFilter != "" || archFilter != "" {
			// TODO: filter by platform JSON in revision
		}

		skills = append(skills, skill)
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"skills": skills,
		"total":  len(skills),
		"limit":  limit,
		"offset": offset,
	})
}
