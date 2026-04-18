-- name: CreateToken :one
INSERT INTO tokens (namespace_id, token_hash, label)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTokenByHash :one
SELECT t.*, n.name as namespace_name, n.type as namespace_type, n.banned as namespace_banned
FROM tokens t
LEFT JOIN namespaces n ON t.namespace_id = n.id
WHERE t.token_hash = $1;

-- name: ListTokensByNamespace :many
SELECT * FROM tokens WHERE namespace_id = $1 ORDER BY created_at DESC;

-- name: DeleteToken :exec
DELETE FROM tokens WHERE id = $1 AND namespace_id = $2;

-- name: IncrementDailyUses :exec
UPDATE tokens SET daily_uses = daily_uses + 1, last_used = NOW() WHERE id = $1;

-- name: ResetDailyUses :execrows
UPDATE tokens SET daily_uses = 0;

-- name: BindTokenToNamespace :exec
UPDATE tokens SET namespace_id = $2 WHERE id = $1;
