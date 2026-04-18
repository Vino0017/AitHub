-- name: AddOrgMember :exec
INSERT INTO org_members (org_id, member_id, role)
VALUES ($1, $2, $3);

-- name: RemoveOrgMember :exec
DELETE FROM org_members WHERE org_id = $1 AND member_id = $2;

-- name: ListOrgMembers :many
SELECT om.*, n.name as member_name
FROM org_members om
JOIN namespaces n ON om.member_id = n.id
WHERE om.org_id = $1
ORDER BY om.role, om.joined_at;

-- name: GetOrgMembership :one
SELECT * FROM org_members WHERE org_id = $1 AND member_id = $2;

-- name: CountOrgOwners :one
SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND role = 'owner';

-- name: IsMemberOfOrg :one
SELECT EXISTS(SELECT 1 FROM org_members WHERE org_id = $1 AND member_id = $2) AS is_member;

-- name: ListUserOrgs :many
SELECT n.*, om.role
FROM org_members om
JOIN namespaces n ON om.org_id = n.id
WHERE om.member_id = $1;
