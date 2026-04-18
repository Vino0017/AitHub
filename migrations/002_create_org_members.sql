-- +goose Up
CREATE TABLE org_members (
    org_id    UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (org_id, member_id)
);

CREATE INDEX idx_org_members_member ON org_members(member_id);

-- +goose Down
DROP TABLE IF EXISTS org_members;
