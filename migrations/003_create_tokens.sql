-- +goose Up
CREATE TABLE tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace_id UUID REFERENCES namespaces(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL,
    label        TEXT NOT NULL DEFAULT '',
    daily_uses   INT NOT NULL DEFAULT 0,
    last_used    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tokens_hash_unique UNIQUE (token_hash)
);

CREATE INDEX idx_tokens_namespace ON tokens(namespace_id) WHERE namespace_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS tokens;
