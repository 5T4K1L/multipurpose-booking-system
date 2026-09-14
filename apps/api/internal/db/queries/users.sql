-- Returns a user by their database ID.
-- name: GetUserByID :one
SELECT
    id,
    email,
    username,
    created_at,
    updated_at
FROM users
WHERE id = $1;

-- Returns a user by their email address.
-- name: GetUserByEmail :one
SELECT
    id,
    email,
    username,
    created_at,
    updated_at
FROM users
WHERE email = $1;