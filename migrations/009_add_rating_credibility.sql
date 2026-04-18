-- +goose Up
-- Add credibility fields to ratings table
ALTER TABLE ratings ADD COLUMN confidence_score FLOAT DEFAULT 1.0;
ALTER TABLE ratings ADD COLUMN execution_time_ms INT;
ALTER TABLE ratings ADD COLUMN error_details JSONB;
ALTER TABLE ratings ADD COLUMN context_metadata JSONB;

-- Add credibility tracking to skills
ALTER TABLE skills ADD COLUMN credibility_score FLOAT DEFAULT 1.0;

-- Create rating patterns table for anomaly detection
CREATE TABLE rating_patterns (
    token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    rating_count INT DEFAULT 0,
    avg_score FLOAT DEFAULT 0,
    score_variance FLOAT DEFAULT 0,
    last_rating_at TIMESTAMPTZ,
    suspicious_flags JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (token_id, skill_id)
);

CREATE INDEX idx_rating_patterns_suspicious ON rating_patterns USING GIN (suspicious_flags);
CREATE INDEX idx_rating_patterns_token ON rating_patterns(token_id);

-- +goose Down
DROP TABLE IF EXISTS rating_patterns;
ALTER TABLE skills DROP COLUMN IF EXISTS credibility_score;
ALTER TABLE ratings DROP COLUMN IF EXISTS context_metadata;
ALTER TABLE ratings DROP COLUMN IF EXISTS error_details;
ALTER TABLE ratings DROP COLUMN IF EXISTS execution_time_ms;
ALTER TABLE ratings DROP COLUMN IF EXISTS confidence_score;
