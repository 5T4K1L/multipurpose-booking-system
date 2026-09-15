-- Returns an OAuth identity by provider and provider subject.
-- name: GetOAuthIdentity :one
SELECT
    id,
    user_id,
    provider,
    provider_subject,
    created_at,
    updated_at
FROM user_oauth_identities
WHERE provider = $1
  AND provider_subject = $2;

-- Returns an OAuth identity for a user and provider.
-- name: GetUserOAuthIdentity :one
SELECT
    id,
    user_id,
    provider,
    provider_subject,
    created_at,
    updated_at
FROM user_oauth_identities
WHERE user_id = $1
  AND provider = $2;

-- Creates an OAuth identity.
-- name: CreateOAuthIdentity :one
INSERT INTO user_oauth_identities (
    user_id,
    provider,
    provider_subject
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING
    id,
    user_id,
    provider,
    provider_subject,
    created_at,
    updated_at;

-- Deletes an OAuth identity.
-- name: DeleteOAuthIdentity :exec
DELETE FROM user_oauth_identities
WHERE id = $1
  AND user_id = $2;