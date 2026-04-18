-- +goose Up
CREATE TABLE revisions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id           UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version            TEXT NOT NULL,
    content            TEXT NOT NULL,
    change_summary     TEXT NOT NULL DEFAULT '',
    author_token_id    UUID REFERENCES tokens(id) ON DELETE SET NULL,
    review_status      TEXT NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending', 'approved', 'revision_requested', 'rejected')),
    review_feedback    JSONB,
    review_result      JSONB,
    review_retry_count INT NOT NULL DEFAULT 0,
    schema_type        TEXT NOT NULL DEFAULT 'skill-md' CHECK (schema_type IN ('skill-md', 'mcp-tool')),
    triggers           TEXT[] NOT NULL DEFAULT '{}',
    compatible_models  TEXT[] NOT NULL DEFAULT '{}',
    estimated_tokens   INT NOT NULL DEFAULT 0,
    requirements       JSONB,
    platform           JSONB,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT revisions_skill_version_unique UNIQUE (skill_id, version)
);

CREATE INDEX idx_revisions_skill ON revisions(skill_id);
CREATE INDEX idx_revisions_status ON revisions(review_status);
CREATE INDEX idx_revisions_created ON revisions(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS revisions;
