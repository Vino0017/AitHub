-- name: SearchSkills :many
SELECT s.*, n.name as namespace_name
FROM skills s
JOIN namespaces n ON s.namespace_id = n.id
WHERE s.status = 'active'
  AND (
    @query::text = '' OR
    to_tsvector('english', s.name || ' ' || s.description || ' ' || array_to_string(s.tags, ' '))
    @@ plainto_tsquery('english', @query::text)
  )
  AND (@framework::text = '' OR s.framework = @framework)
  AND (@tag::text = '' OR @tag = ANY(s.tags))
  AND (
    s.visibility = 'public'
    OR (s.visibility = 'private' AND s.namespace_id = @caller_namespace_id)
    OR (s.visibility = 'org' AND EXISTS (
      SELECT 1 FROM org_members om
      WHERE om.org_id = s.namespace_id AND om.member_id = @caller_namespace_id
    ))
  )
ORDER BY
  CASE WHEN @sort::text = 'rating' THEN s.avg_rating END DESC,
  CASE WHEN @sort::text = 'installs' THEN s.install_count END DESC,
  CASE WHEN @sort::text = 'recent' OR @sort::text = 'new' THEN EXTRACT(EPOCH FROM s.created_at) END DESC,
  s.avg_rating DESC
LIMIT @lim OFFSET @off;

-- name: GetSkillByNamespaceName :one
SELECT s.*, n.name as namespace_name
FROM skills s
JOIN namespaces n ON s.namespace_id = n.id
WHERE n.name = $1 AND s.name = $2;

-- name: GetSkillByID :one
SELECT s.*, n.name as namespace_name
FROM skills s
JOIN namespaces n ON s.namespace_id = n.id
WHERE s.id = $1;

-- name: CreateSkill :one
INSERT INTO skills (namespace_id, name, description, tags, framework, visibility, forked_from, latest_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: IncrementInstallCount :exec
UPDATE skills SET install_count = install_count + 1 WHERE id = $1;

-- name: IncrementForkCount :exec
UPDATE skills SET fork_count = fork_count + 1 WHERE id = $1;

-- name: UpdateSkillLatestVersion :exec
UPDATE skills SET latest_version = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateSkillStatus :exec
UPDATE skills SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateSkillRating :exec
UPDATE skills SET avg_rating = $2, rating_count = $3, outcome_success_rate = $4, updated_at = NOW() WHERE id = $1;

-- name: ListSkillsByNamespace :many
SELECT s.*, n.name as namespace_name
FROM skills s
JOIN namespaces n ON s.namespace_id = n.id
WHERE s.namespace_id = $1 AND s.status = 'active'
ORDER BY s.updated_at DESC;

-- name: ListForks :many
SELECT s.*, n.name as namespace_name
FROM skills s
JOIN namespaces n ON s.namespace_id = n.id
WHERE s.forked_from = $1 AND s.status = 'active'
ORDER BY s.install_count DESC;

-- name: GetTrendingSkills :many
SELECT s.*, n.name as namespace_name
FROM skills s
JOIN namespaces n ON s.namespace_id = n.id
WHERE s.status = 'active' AND s.visibility = 'public'
  AND s.created_at > NOW() - INTERVAL '30 days'
ORDER BY s.install_count DESC
LIMIT $1;
