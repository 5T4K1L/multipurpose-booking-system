-- Returns the password credential for a user.
-- name: GetUserCredential :one
SELECT
    id,
    user_id,
    password_hash,
    created_at,
    updated_at
FROM user_credentials
WHERE user_id = $1;

-- Creates or stores a user's password credential.
-- name: CreateUserCredential :one
INSERT INTO user_credentials (
    user_id,
    password_hash
)
VALUES (
    $1,
    $2
)
RETURNING
    id,
    user_id,
    password_hash,
    created_at,
    updated_at;

-- Updates an existing password hash.
-- name: UpdateUserPasswordHash :exec
UPDATE user_credentials
SET
    password_hash = $2,
    updated_at = NOW()
WHERE user_id = $1;

-- Creates a new authentication session.
-- name: CreateUserSession :one
INSERT INTO user_sessions (
    user_id,
    token_hash,
    expires_at
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING
    id,
    user_id,
    token_hash,
    expires_at,
    created_at,
    last_seen_at,
    revoked_at;

-- Returns a session by its token hash.
-- name: GetUserSessionByTokenHash :one
SELECT
    id,
    user_id,
    token_hash,
    expires_at,
    created_at,
    last_seen_at,
    revoked_at
FROM user_sessions
WHERE token_hash = $1;

-- Updates the most recent session activity.
-- name: UpdateUserSessionLastSeen :exec
UPDATE user_sessions
SET
    last_seen_at = NOW()
WHERE id = $1;

-- Revokes a session.
-- name: RevokeUserSession :exec
UPDATE user_sessions
SET
    revoked_at = NOW()
WHERE id = $1
  AND revoked_at IS NULL;

-- Revokes all sessions belonging to a user.
-- name: RevokeAllUserSessions :exec
UPDATE user_sessions
SET
    revoked_at = NOW()
WHERE user_id = $1
  AND revoked_at IS NULL;