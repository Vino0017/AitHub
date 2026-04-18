-- +goose Up
CREATE TABLE skills (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace_id         UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    name                 TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    tags                 TEXT[] NOT NULL DEFAULT '{}',
    framework            TEXT NOT NULL DEFAULT '',
    visibility           TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private', 'org')),
    forked_from          UUID REFERENCES skills(id) ON DELETE SET NULL,
    install_count        INT NOT NULL DEFAULT 0,
    avg_rating           NUMERIC(4,2) NOT NULL DEFAULT 0,
    rating_count         INT NOT NULL DEFAULT 0,
    outcome_success_rate NUMERIC(4,3) NOT NULL DEFAULT 0,
    latest_version       TEXT NOT NULL DEFAULT '',
    fork_count           INT NOT NULL DEFAULT 0,
    status               TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'yanked', 'removed')),
    search_vector        tsvector,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT skills_namespace_name_unique UNIQUE (namespace_id, name),
    CONSTRAINT skills_name_format CHECK (name ~ '^[a-z0-9][a-z0-9-]{1,98}[a-z0-9]$')
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION skills_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('simple',
        coalesce(NEW.name, '') || ' ' ||
        coalesce(NEW.description, '') || ' ' ||
        coalesce(array_to_string(NEW.tags, ' '), '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_skills_search_vector
    BEFORE INSERT OR UPDATE ON skills
    FOR EACH ROW EXECUTE FUNCTION skills_search_vector_update();

CREATE INDEX idx_skills_search ON skills USING gin(search_vector);
CREATE INDEX idx_skills_framework ON skills(framework) WHERE framework != '';
CREATE INDEX idx_skills_visibility ON skills(visibility);
CREATE INDEX idx_skills_status ON skills(status);
CREATE INDEX idx_skills_forked_from ON skills(forked_from) WHERE forked_from IS NOT NULL;
CREATE INDEX idx_skills_rating ON skills(avg_rating DESC) WHERE status = 'active';
CREATE INDEX idx_skills_installs ON skills(install_count DESC) WHERE status = 'active';
CREATE INDEX idx_skills_created ON skills(created_at DESC) WHERE status = 'active';

-- +goose Down
DROP TRIGGER IF EXISTS trg_skills_search_vector ON skills;
DROP FUNCTION IF EXISTS skills_search_vector_update;
DROP TABLE IF EXISTS skills;
