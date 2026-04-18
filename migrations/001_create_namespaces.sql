-- +goose Up
CREATE EXTENSION IF NOT EXISTS "vector";

CREATE TABLE namespaces (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'personal' CHECK (type IN ('personal', 'org')),
    github_id  TEXT,
    google_id  TEXT,
    email      TEXT,
    banned     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT namespaces_name_unique UNIQUE (name),
    CONSTRAINT namespaces_name_format CHECK (name ~ '^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$')
);

CREATE INDEX idx_namespaces_github_id ON namespaces(github_id) WHERE github_id IS NOT NULL;
CREATE INDEX idx_namespaces_google_id ON namespaces(google_id) WHERE google_id IS NOT NULL;
CREATE INDEX idx_namespaces_email ON namespaces(email) WHERE email IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS namespaces;
