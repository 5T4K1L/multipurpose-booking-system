-- Creates the core platform identity table.
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,

    -- Primary identity email.
    email TEXT NOT NULL,

    -- Display name for the user.
    username TEXT NOT NULL,

    -- Record creation time.
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Record update time.
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Prevent duplicate email identities.
    CONSTRAINT users_email_unique UNIQUE (email)
);