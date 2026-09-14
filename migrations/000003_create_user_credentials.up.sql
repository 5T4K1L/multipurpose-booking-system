-- Creates the password credential table for platform users.
CREATE TABLE user_credentials (
    id BIGSERIAL PRIMARY KEY,

    -- Links the credential to the platform identity.
    user_id BIGINT NOT NULL,

    -- Stores only the password hash, never the plaintext password.
    password_hash TEXT NOT NULL,

    -- Tracks when the credential was created.
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Tracks when the credential was last changed.
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Each user has one password credential in the initial model.
    CONSTRAINT user_credentials_user_id_unique UNIQUE (user_id),

    -- Protects the relationship with the users table.
    CONSTRAINT user_credentials_user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES users (id)
);