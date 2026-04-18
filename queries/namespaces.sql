-- name: GetNamespaceByName :one
SELECT * FROM namespaces WHERE name = $1;

-- name: GetNamespaceByID :one
SELECT * FROM namespaces WHERE id = $1;

-- name: GetNamespaceByGitHubID :one
SELECT * FROM namespaces WHERE github_id = $1;

-- name: GetNamespaceByGoogleID :one
SELECT * FROM namespaces WHERE google_id = $1;

-- name: GetNamespaceByEmail :one
SELECT * FROM namespaces WHERE email = $1;

-- name: CreateNamespace :one
INSERT INTO namespaces (name, type, github_id, google_id, email)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: BanNamespace :exec
UPDATE namespaces SET banned = TRUE WHERE id = $1;
