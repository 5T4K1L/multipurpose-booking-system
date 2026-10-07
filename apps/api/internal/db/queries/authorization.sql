-- name: GetAuthorizationState :many
-- One statement gives a current account/role snapshot, never session-cached roles.
SELECT users.status, COALESCE(user_platform_roles.role, '')::text AS role
FROM users
LEFT JOIN user_platform_roles ON user_platform_roles.user_id = users.id
WHERE users.id = $1;
