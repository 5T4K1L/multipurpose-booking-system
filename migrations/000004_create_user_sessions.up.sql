-- Creates the server-side authentication session table.
CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,

    -- Links the session to the authenticated user.
    user_id BIGINT NOT NULL,

    -- Stores a hash of the session token.
    token_hash TEXT NOT NULL,

    -- Determines when the session becomes invalid.
    expires_at TIMESTAMP NOT NULL,

    -- Tracks when the session was created.
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Tracks the most recent session activity when applicable.
    last_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Marks a session as explicitly revoked.
    revoked_at TIMESTAMP,

    -- Session token hashes must be unique.
    CONSTRAINT user_sessions_token_hash_unique UNIQUE (token_hash),

    -- Protects the relationship with the users table.
    CONSTRAINT user_sessions_user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES users (id)
);

-- Speeds up session lookups by user.
CREATE INDEX idx_user_sessions_user_id
    ON user_sessions (user_id);

-- Supports expiration-related session queries.
CREATE INDEX idx_user_sessions_expires_at
    ON user_sessions (expires_at);