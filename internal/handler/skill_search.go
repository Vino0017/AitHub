package handler

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
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

// Query expansion: map common terms to their variants
var queryExpansion = map[string][]string{
	"k8s":        {"kubernetes", "k8s"},
	"kubernetes": {"kubernetes", "k8s"},
	"deploy":     {"deploy", "deployment", "deploying"},
	"review":     {"review", "audit", "check"},
	"test":       {"test", "testing", "qa"},
	"docker":     {"docker", "container", "containerize"},
	"ci":         {"ci", "continuous integration", "pipeline"},
	"cd":         {"cd", "continuous deployment", "delivery"},
}

// expandQuery expands a query with synonyms
func expandQuery(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	expanded := make(map[string]bool)

	for _, word := range words {
		expanded[word] = true
		if variants, ok := queryExpansion[word]; ok {
			for _, v := range variants {
				expanded[v] = true
			}
		}
	}

	result := make([]string, 0, len(expanded))
	for word := range expanded {
		result = append(result, word)
	}
	return result
}

// Search handles GET /v1/skills with enhanced relevance
func (h *SkillSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	framework := q.Get("framework")
	tag := q.Get("tag")
	sort := q.Get("sort")
	osFilter := q.Get("os")
	archFilter := q.Get("arch")
	explore := q.Get("explore") == "true"

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	if sort == "" {
		sort = "rating"
	}

	// E&E: 20% explore mode (new skills)
	if explore && rand.Float64() < 0.2 {
		sort = "new"
	}

	// Build the caller's namespace ID for visibility filtering
	var callerNsID interface{}
	if !middleware.IsAnonymous(r.Context()) {
		callerNsID = middleware.GetNamespaceID(r.Context())
	}

	// Expand query for better matching
	expandedTerms := expandQuery(query)
	expandedQuery := strings.Join(expandedTerms, " | ")

	// Build ORDER BY with hybrid scoring
	orderBy := "s.avg_rating DESC"
	switch sort {
	case "installs":
		orderBy = "s.install_count DESC"
	case "recent", "new":
		orderBy = "s.created_at DESC"
	case "trending":
		orderBy = "s.install_count DESC"
	case "rating":
		// Hybrid scoring: rating * success_rate with time boost for new skills
		orderBy = `(s.avg_rating * s.outcome_success_rate *
		           (1 + 0.1 * LEAST(EXTRACT(EPOCH FROM (NOW() - s.created_at)) / 86400 / 30, 1))) DESC`
	}

	// Build WHERE clauses
	sqlQuery := `
		SELECT s.id, s.namespace_id, n.name AS namespace_name, s.name, s.description,
		       s.tags, s.framework, s.visibility, s.forked_from,
		       s.install_count, s.avg_rating, s.rating_count, s.outcome_success_rate,
		       s.latest_version, s.fork_count, s.status, s.created_at, s.updated_at,
		       (SELECT r.triggers FROM revisions r
		        WHERE r.skill_id = s.id AND r.review_status = 'approved'
		        ORDER BY r.created_at DESC LIMIT 1) AS triggers,
		       CASE WHEN s.created_at > NOW() - INTERVAL '7 days' THEN true ELSE false END AS is_new
		FROM skills s
		JOIN namespaces n ON s.namespace_id = n.id
		WHERE s.status = 'active'`

	args := []interface{}{}
	argIdx := 1

	if query != "" {
		// Priority 1: Triggers match (highest relevance)
		// Priority 2: Full-text search
		sqlQuery += ` AND (
			EXISTS (
				SELECT 1 FROM revisions r
				WHERE r.skill_id = s.id
				AND r.review_status = 'approved'
				AND r.triggers && $` + strconv.Itoa(argIdx) + `::text[]
			)
			OR s.search_vector @@ to_tsquery('simple', $` + strconv.Itoa(argIdx+1) + `)
		)`
		// Convert []string to pgtype.Array for PostgreSQL
		textArray := &pgtype.Array[string]{}
		textArray.Elements = make([]string, len(expandedTerms))
		copy(textArray.Elements, expandedTerms)
		textArray.Dims = []pgtype.ArrayDimension{{Length: int32(len(expandedTerms)), LowerBound: 1}}
		textArray.Valid = true

		args = append(args, textArray, expandedQuery)
		argIdx += 2
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
			triggers                                []string
			forkedFrom                              interface{}
			installs, ratingCount, forkCount         int
			avgRating, successRate                   float64
			createdAt, updatedAt                     interface{}
			isNew                                    bool
		)
		err := rows.Scan(&id, &nsID, &nsName, &name, &desc, &tags, &fw, &vis, &forkedFrom,
			&installs, &avgRating, &ratingCount, &successRate, &lv, &forkCount, &status, &createdAt, &updatedAt,
			&triggers, &isNew)
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
			"is_new":               isNew,
		}

		// Platform filtering (post-query for now, can optimize with JSON query later)
		if osFilter != "" || archFilter != "" {
			// TODO: filter by platform JSON in revision
		}

		skills = append(skills, skill)
	}

	// Get total count for pagination
	totalCount := h.getTotalCount(r.Context(), query, framework, tag, sort, callerNsID)

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"skills": skills,
		"total":  totalCount,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *SkillSearchHandler) getTotalCount(ctx context.Context, query, framework, tag, sort string, callerNsID interface{}) int {
	expandedTerms := expandQuery(query)
	expandedQuery := strings.Join(expandedTerms, " | ")

	sqlQuery := `SELECT COUNT(*) FROM skills s WHERE s.status = 'active'`
	args := []interface{}{}
	argIdx := 1

	if query != "" {
		sqlQuery += ` AND (
			EXISTS (
				SELECT 1 FROM revisions r
				WHERE r.skill_id = s.id
				AND r.review_status = 'approved'
				AND r.triggers && ARRAY[$` + strconv.Itoa(argIdx) + `]::text[]
			)
			OR s.search_vector @@ to_tsquery('simple', $` + strconv.Itoa(argIdx+1) + `)
		)`
		args = append(args, expandedTerms, expandedQuery)
		argIdx += 2
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

	if callerNsID != nil {
		sqlQuery += ` AND (s.visibility = 'public' OR (s.visibility = 'private' AND s.namespace_id = $` + strconv.Itoa(argIdx) + `)
		              OR (s.visibility = 'org' AND EXISTS (
		                SELECT 1 FROM org_members om WHERE om.org_id = s.namespace_id AND om.member_id = $` + strconv.Itoa(argIdx) + `)))`
		args = append(args, callerNsID)
	} else {
		sqlQuery += ` AND s.visibility = 'public'`
	}

	var count int
	h.pool.QueryRow(ctx, sqlQuery, args...).Scan(&count)
	return count
}
