-- name: UpsertRating :one
INSERT INTO ratings (skill_id, revision_id, token_id, score, outcome, task_type, model_used, tokens_consumed, failure_reason)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (revision_id, token_id) DO UPDATE SET
    score = EXCLUDED.score,
    outcome = EXCLUDED.outcome,
    task_type = EXCLUDED.task_type,
    model_used = EXCLUDED.model_used,
    tokens_consumed = EXCLUDED.tokens_consumed,
    failure_reason = EXCLUDED.failure_reason,
    updated_at = NOW()
RETURNING *;

-- name: GetSkillRatingStats :one
-- Only counts ratings from registered (non-anonymous) tokens for the latest revision
SELECT
    COUNT(*)::int AS rating_count,
    COALESCE(AVG(r.score), 0)::float AS avg_score,
    COALESCE(SUM(CASE WHEN r.outcome = 'success' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0)::float AS success_rate
FROM ratings r
JOIN tokens t ON r.token_id = t.id
WHERE r.revision_id = $1 AND t.namespace_id IS NOT NULL;

-- name: RefreshSkillRatings :exec
-- Recalculate avg_rating based on latest approved revision, registered tokens only
UPDATE skills s SET
    avg_rating = sub.bayesian_avg,
    rating_count = sub.n,
    outcome_success_rate = sub.success_rate,
    updated_at = NOW()
FROM (
    SELECT
        sk.id AS skill_id,
        COALESCE(stats.n, 0) AS n,
        CASE WHEN COALESCE(stats.n, 0) = 0 THEN 0
             ELSE (5.0 * 6.0 + COALESCE(stats.total_score, 0)) / (5.0 + COALESCE(stats.n, 0))
        END AS bayesian_avg,
        COALESCE(stats.success_rate, 0) AS success_rate
    FROM skills sk
    LEFT JOIN LATERAL (
        SELECT rev.id AS rev_id
        FROM revisions rev
        WHERE rev.skill_id = sk.id AND rev.review_status = 'approved'
        ORDER BY rev.created_at DESC LIMIT 1
    ) latest_rev ON TRUE
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*)::int AS n,
            SUM(r.score)::float AS total_score,
            SUM(CASE WHEN r.outcome = 'success' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) AS success_rate
        FROM ratings r
        JOIN tokens t ON r.token_id = t.id
        WHERE r.revision_id = latest_rev.rev_id AND t.namespace_id IS NOT NULL
    ) stats ON TRUE
    WHERE sk.status = 'active'
) sub
WHERE s.id = sub.skill_id;
