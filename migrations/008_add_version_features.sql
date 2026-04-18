-- +goose Up
ALTER TABLE revisions ADD COLUMN breaking_change BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE revisions ADD COLUMN migration_guide TEXT;

CREATE INDEX idx_revisions_breaking ON revisions(breaking_change) WHERE breaking_change = true;

-- +goose Down
DROP INDEX IF EXISTS idx_revisions_breaking;
ALTER TABLE revisions DROP COLUMN IF EXISTS migration_guide;
ALTER TABLE revisions DROP COLUMN IF EXISTS breaking_change;
