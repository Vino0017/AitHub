package handler

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/embedding"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type SkillSearchHandler struct {
	pool    *pgxpool.Pool
	embedCl *embedding.Client
}

func NewSkillSearchHandler(pool *pgxpool.Pool, embedCl *embedding.Client) *SkillSearchHandler {
	return &SkillSearchHandler{pool: pool, embedCl: embedCl}
}

// Query expansion: map common terms to their variants
var queryExpansion = map[string][]string{
	"k8s":        {"kubernetes", "k8s"},
	"kubernetes": {"kubernetes", "k8s"},
	"deploy":     {"deploy", "deployment", "deploying"},
	"review":     {"review", "audit", "check"},
	"test":       {"test", "testing", "qa"},
	"docker":     {"docker", "container", "containerize"},
	"ci":         {"ci", "pipeline"},
	"cd":         {"cd", "delivery"},
	"blog":       {"blog", "website", "cms"},
	"api":        {"api", "rest", "endpoint"},
	"db":         {"db", "database", "sql"},
	"database":   {"database", "db", "sql"},
	"ml":         {"ml", "machine", "learning"},
	"ai":         {"ai", "artificial", "intelligence"},
}

// expandQuery expands a query with single-word synonyms
func expandQuery(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	expanded := make(map[string]bool)

	for _, word := range words {
		// Strip non-alphanumeric for tsquery safety
		clean := sanitizeForTsquery(word)
		if clean != "" {
			expanded[clean] = true
		}
		if variants, ok := queryExpansion[clean]; ok {
			for _, v := range variants {
				// Only add single-word variants (no spaces)
				if !strings.Contains(v, " ") {
					expanded[v] = true
				}
			}
		}
	}

	result := make([]string, 0, len(expanded))
	for word := range expanded {
		result = append(result, word)
	}
	return result
}

// sanitizeForTsquery removes characters that break to_tsquery
func sanitizeForTsquery(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// hasNonASCII checks if string contains non-ASCII characters (e.g., CJK)
func hasNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

// Search handles GET /v1/skills with hybrid text + vector search
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

	// Try vector search if query is present and embeddings are configured
	var queryEmbedding []float32
	useVectorSearch := false
	if query != "" && h.embedCl != nil && h.embedCl.IsConfigured() {
		if emb, err := h.embedCl.Embed(r.Context(), query); err == nil {
			queryEmbedding = emb
			useVectorSearch = true
		} else {
			log.Printf("[search] embedding failed, falling back to text search: %v", err)
		}
	}

	// Expand query for text matching fallback
	expandedTerms := expandQuery(query)
	// For tsquery: join single sanitized words with | operator
	expandedQuery := strings.Join(expandedTerms, " | ")
	// Check if query has non-ASCII (CJK, etc.) - use ILIKE fallback
	nonASCII := hasNonASCII(query)

	// Build ORDER BY
	orderBy := "s.avg_rating DESC"
	switch sort {
	case "installs":
		orderBy = "s.install_count DESC"
	case "recent", "new":
		orderBy = "s.created_at DESC"
	case "trending":
		orderBy = "s.install_count DESC"
	case "rating":
		if useVectorSearch {
			// Hybrid ranking: 60% semantic + 20% rating + 10% popularity + 10% success
			orderBy = "hybrid_score DESC"
		} else {
			orderBy = `(s.avg_rating * s.outcome_success_rate *
			           (1 + 0.1 * LEAST(EXTRACT(EPOCH FROM (NOW() - s.created_at)) / 86400 / 30, 1))) DESC`
		}
	}

	// Build query
	var selectExtra, whereExtra string
	args := []interface{}{}
	argIdx := 1

	if useVectorSearch {
		// Hybrid: similarity * (base + quality_boost)
		// Quality can only amplify relevance, never override it
		selectExtra = fmt.Sprintf(`,
			CASE WHEN s.embedding IS NOT NULL
			     THEN 1 - (s.embedding <=> $%d::vector)
			     ELSE 0
			END AS similarity,
			CASE WHEN s.embedding IS NOT NULL
			     THEN (1 - (s.embedding <=> $%d::vector))
			          * (0.7
			             + 0.15 * (s.avg_rating / 5.0)
			             + 0.10 * LEAST(LN(s.install_count + 1) / 10.0, 1.0)
			             + 0.05 * s.outcome_success_rate)
			     ELSE 0
			END AS hybrid_score`, argIdx, argIdx)
		// Convert []float32 to pgvector string
		vecStr := float32SliceToVectorString(queryEmbedding)
		args = append(args, vecStr)
		argIdx++

		// Match: vector similarity OR text search
		if query != "" {
			if nonASCII || len(expandedTerms) == 0 {
				// Non-ASCII or empty terms: use vector + ILIKE only
				whereExtra = fmt.Sprintf(` AND (
					(s.embedding IS NOT NULL AND 1 - (s.embedding <=> $%d::vector) > 0.5)
					OR s.name ILIKE $%d
					OR s.description ILIKE $%d
				)`, argIdx-1, argIdx, argIdx)
				args = append(args, "%"+query+"%")
				argIdx++
			} else {
				whereExtra = fmt.Sprintf(` AND (
					(s.embedding IS NOT NULL AND 1 - (s.embedding <=> $%d::vector) > 0.5)
					OR EXISTS (
						SELECT 1 FROM revisions r
						WHERE r.skill_id = s.id
						AND r.review_status = 'approved'
						AND r.triggers && $%d::text[]
					)
					OR s.search_vector @@ plainto_tsquery('simple', $%d)
					OR s.name ILIKE $%d
					OR s.description ILIKE $%d
				)`, argIdx-1, argIdx, argIdx+1, argIdx+2, argIdx+2)
				textArray := &pgtype.Array[string]{}
				textArray.Elements = make([]string, len(expandedTerms))
				copy(textArray.Elements, expandedTerms)
				textArray.Dims = []pgtype.ArrayDimension{{Length: int32(len(expandedTerms)), LowerBound: 1}}
				textArray.Valid = true
				args = append(args, textArray, expandedQuery, "%"+query+"%")
				argIdx += 3
			}
		}
	} else {
		selectExtra = ", 0::float AS similarity, 0::float AS hybrid_score"
		if query != "" {
			if nonASCII || len(expandedTerms) == 0 {
				// Non-ASCII: ILIKE only
				whereExtra = fmt.Sprintf(` AND (
					s.name ILIKE $%d
					OR s.description ILIKE $%d
				)`, argIdx, argIdx)
				args = append(args, "%"+query+"%")
				argIdx++
			} else {
				whereExtra = fmt.Sprintf(` AND (
					EXISTS (
						SELECT 1 FROM revisions r
						WHERE r.skill_id = s.id
						AND r.review_status = 'approved'
						AND r.triggers && $%d::text[]
					)
					OR s.search_vector @@ plainto_tsquery('simple', $%d)
					OR s.name ILIKE $%d
					OR s.description ILIKE $%d
				)`, argIdx, argIdx+1, argIdx+2, argIdx+2)
				textArray := &pgtype.Array[string]{}
				textArray.Elements = make([]string, len(expandedTerms))
				copy(textArray.Elements, expandedTerms)
				textArray.Dims = []pgtype.ArrayDimension{{Length: int32(len(expandedTerms)), LowerBound: 1}}
				textArray.Valid = true
				args = append(args, textArray, expandedQuery, "%"+query+"%")
				argIdx += 3
			}
		}
	}

	if framework != "" {
		whereExtra += fmt.Sprintf(` AND s.framework = $%d`, argIdx)
		args = append(args, framework)
		argIdx++
	}

	if tag != "" {
		whereExtra += fmt.Sprintf(` AND $%d = ANY(s.tags)`, argIdx)
		args = append(args, tag)
		argIdx++
	}

	if sort == "new" {
		whereExtra += ` AND s.created_at > NOW() - INTERVAL '7 days'`
	}

	// Visibility filter
	if callerNsID != nil {
		whereExtra += fmt.Sprintf(` AND (s.visibility = 'public' OR (s.visibility = 'private' AND s.namespace_id = $%d)
		              OR (s.visibility = 'org' AND EXISTS (
		                SELECT 1 FROM org_members om WHERE om.org_id = s.namespace_id AND om.member_id = $%d)))`, argIdx, argIdx)
		args = append(args, callerNsID)
		argIdx++
	} else {
		whereExtra += ` AND s.visibility = 'public'`
	}

	sqlQuery := fmt.Sprintf(`
		SELECT s.id, s.namespace_id, n.name AS namespace_name, s.name, s.description,
		       s.tags, s.framework, s.visibility, s.forked_from,
		       s.install_count, s.avg_rating, s.rating_count, s.outcome_success_rate,
		       s.latest_version, s.fork_count, s.status, s.created_at, s.updated_at,
		       (SELECT r.triggers FROM revisions r
		        WHERE r.skill_id = s.id AND r.review_status = 'approved'
		        ORDER BY r.created_at DESC LIMIT 1) AS triggers,
		       CASE WHEN s.created_at > NOW() - INTERVAL '7 days' THEN true ELSE false END AS is_new
		       %s
		FROM skills s
		JOIN namespaces n ON s.namespace_id = n.id
		WHERE s.status = 'active' %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		selectExtra, whereExtra, orderBy, argIdx, argIdx+1)
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
			installs, ratingCount, forkCount             int
			avgRating, successRate, similarity, hybridScore float64
			createdAt, updatedAt                     interface{}
			isNew                                    bool
		)
		err := rows.Scan(&id, &nsID, &nsName, &name, &desc, &tags, &fw, &vis, &forkedFrom,
			&installs, &avgRating, &ratingCount, &successRate, &lv, &forkCount, &status, &createdAt, &updatedAt,
			&triggers, &isNew, &similarity, &hybridScore)
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

		if useVectorSearch && similarity > 0 {
			skill["relevance_score"] = hybridScore
			skill["semantic_score"] = similarity
		}

		// Platform filtering (post-query for now)
		if osFilter != "" || archFilter != "" {
			// TODO: filter by platform JSON in revision
		}

		skills = append(skills, skill)
	}

	// Get total count for pagination
	totalCount := h.getTotalCount(r.Context(), query, framework, tag, sort, callerNsID)

	response := map[string]interface{}{
		"skills": skills,
		"total":  totalCount,
		"limit":  limit,
		"offset": offset,
	}
	if useVectorSearch {
		response["search_mode"] = "semantic"
	} else if query != "" {
		response["search_mode"] = "text"
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// float32SliceToVectorString converts []float32 to pgvector string format "[0.1,0.2,...]"
func float32SliceToVectorString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (h *SkillSearchHandler) getTotalCount(ctx context.Context, query, framework, tag, sort string, callerNsID interface{}) int {
	sqlQuery := `SELECT COUNT(*) FROM skills s WHERE s.status = 'active'`
	args := []interface{}{}
	argIdx := 1

	if query != "" {
		// For total count, use a simpler text search (vector results are a superset)
		expandedTerms := expandQuery(query)
		expandedQuery := strings.Join(expandedTerms, " | ")

		sqlQuery += fmt.Sprintf(` AND (
			s.search_vector @@ to_tsquery('simple', $%d)
			OR s.embedding IS NOT NULL
		)`, argIdx)
		args = append(args, expandedQuery)
		argIdx++
	}

	if framework != "" {
		sqlQuery += fmt.Sprintf(` AND s.framework = $%d`, argIdx)
		args = append(args, framework)
		argIdx++
	}

	if tag != "" {
		sqlQuery += fmt.Sprintf(` AND $%d = ANY(s.tags)`, argIdx)
		args = append(args, tag)
		argIdx++
	}

	if sort == "new" {
		sqlQuery += ` AND s.created_at > NOW() - INTERVAL '7 days'`
	}

	if callerNsID != nil {
		sqlQuery += fmt.Sprintf(` AND (s.visibility = 'public' OR (s.visibility = 'private' AND s.namespace_id = $%d)
		              OR (s.visibility = 'org' AND EXISTS (
		                SELECT 1 FROM org_members om WHERE om.org_id = s.namespace_id AND om.member_id = $%d)))`, argIdx, argIdx)
		args = append(args, callerNsID)
	} else {
		sqlQuery += ` AND s.visibility = 'public'`
	}

	var count int
	h.pool.QueryRow(ctx, sqlQuery, args...).Scan(&count)
	return count
}

