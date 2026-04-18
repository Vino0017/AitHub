-- +goose Up
-- Email verification codes (temporary, expire after 10 minutes)
CREATE TABLE email_verifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL,
    namespace  TEXT NOT NULL,
    code       TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_verifications_email ON email_verifications(email);

-- OAuth device flow state (temporary, expire after 15 minutes)
CREATE TABLE oauth_device_flows (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        TEXT NOT NULL CHECK (provider IN ('github', 'google')),
    device_code     TEXT NOT NULL,
    user_code       TEXT NOT NULL,
    verification_uri TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    completed       BOOLEAN NOT NULL DEFAULT FALSE,
    namespace_id    UUID REFERENCES namespaces(id),
    token_id        UUID REFERENCES tokens(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_device_code ON oauth_device_flows(device_code);

-- +goose Down
DROP TABLE IF EXISTS oauth_device_flows;
DROP TABLE IF EXISTS email_verifications;
