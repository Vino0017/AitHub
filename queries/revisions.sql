-- name: CreateRevision :one
INSERT INTO revisions (skill_id, version, content, change_summary, author_token_id, review_status, schema_type, triggers, compatible_models, estimated_tokens, requirements, platform)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetRevision :one
SELECT * FROM revisions WHERE id = $1;

-- name: GetRevisionBySkillVersion :one
SELECT * FROM revisions WHERE skill_id = $1 AND version = $2;

-- name: GetLatestApprovedRevision :one
SELECT * FROM revisions
WHERE skill_id = $1 AND review_status = 'approved'
ORDER BY created_at DESC LIMIT 1;

-- name: ListRevisionsBySkill :many
SELECT * FROM revisions
WHERE skill_id = $1
ORDER BY created_at DESC;

-- name: UpdateRevisionStatus :exec
UPDATE revisions SET review_status = $2, review_feedback = $3, review_result = $4 WHERE id = $1;

-- name: IncrementReviewRetryCount :one
UPDATE revisions SET review_retry_count = review_retry_count + 1 WHERE id = $1 RETURNING review_retry_count;

-- name: CountPendingRevisionsBySkill :one
SELECT COUNT(*) FROM revisions WHERE skill_id = $1 AND review_status IN ('pending', 'revision_requested');
