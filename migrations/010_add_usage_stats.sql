-- +goose Up
-- Add usage tracking table
CREATE TABLE skill_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL, -- install | execute | rate
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_usage_logs_skill ON skill_usage_logs(skill_id, created_at DESC);
CREATE INDEX idx_usage_logs_token ON skill_usage_logs(token_id, created_at DESC);
CREATE INDEX idx_usage_logs_action ON skill_usage_logs(action);

-- Add usage stats to skills table
ALTER TABLE skills ADD COLUMN dau INT DEFAULT 0;
ALTER TABLE skills ADD COLUMN mau INT DEFAULT 0;
ALTER TABLE skills ADD COLUMN retention_rate FLOAT DEFAULT 0;
ALTER TABLE skills ADD COLUMN last_used_at TIMESTAMPTZ;
ALTER TABLE skills ADD COLUMN is_zombie BOOLEAN DEFAULT false;

CREATE INDEX idx_skills_zombie ON skills(is_zombie) WHERE is_zombie = true;
CREATE INDEX idx_skills_last_used ON skills(last_used_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_skills_last_used;
DROP INDEX IF EXISTS idx_skills_zombie;
ALTER TABLE skills DROP COLUMN IF EXISTS is_zombie;
ALTER TABLE skills DROP COLUMN IF EXISTS last_used_at;
ALTER TABLE skills DROP COLUMN IF EXISTS retention_rate;
ALTER TABLE skills DROP COLUMN IF EXISTS mau;
ALTER TABLE skills DROP COLUMN IF EXISTS dau;

DROP INDEX IF EXISTS idx_usage_logs_action;
DROP INDEX IF EXISTS idx_usage_logs_token;
DROP INDEX IF EXISTS idx_usage_logs_skill;
DROP TABLE IF EXISTS skill_usage_logs;
