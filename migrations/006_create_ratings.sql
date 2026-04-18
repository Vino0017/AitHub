-- +goose Up
CREATE TABLE ratings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id        UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    revision_id     UUID NOT NULL REFERENCES revisions(id) ON DELETE CASCADE,
    token_id        UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
    score           INT NOT NULL CHECK (score >= 1 AND score <= 10),
    outcome         TEXT NOT NULL DEFAULT 'success' CHECK (outcome IN ('success', 'partial', 'failure')),
    task_type       TEXT NOT NULL DEFAULT '',
    model_used      TEXT NOT NULL DEFAULT '',
    tokens_consumed INT NOT NULL DEFAULT 0,
    failure_reason  TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Same token can only rate same revision once (upsert on conflict)
    CONSTRAINT ratings_revision_token_unique UNIQUE (revision_id, token_id)
);

CREATE INDEX idx_ratings_skill ON ratings(skill_id);
CREATE INDEX idx_ratings_revision ON ratings(revision_id);
CREATE INDEX idx_ratings_token ON ratings(token_id);

-- +goose Down
DROP TABLE IF EXISTS ratings;
