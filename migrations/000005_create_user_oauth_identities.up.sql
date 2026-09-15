-- Stores external OAuth identities linked to platform users.
CREATE TABLE user_oauth_identities (
    id BIGSERIAL PRIMARY KEY,

    -- Links the external identity to the platform user.
    user_id BIGINT NOT NULL,

    -- OAuth provider identifier.
    provider TEXT NOT NULL,

    -- Stable provider-issued identity subject.
    provider_subject TEXT NOT NULL,

    -- Record creation time.
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Record update time.
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Protects the relationship with the users table.
    CONSTRAINT user_oauth_identities_user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES users (id),

    -- A provider subject identifies exactly one external account.
    CONSTRAINT user_oauth_identities_provider_subject_unique
        UNIQUE (provider, provider_subject),

    -- A user can initially have at most one identity per provider.
    CONSTRAINT user_oauth_identities_user_provider_unique
        UNIQUE (user_id, provider),

    -- Only supported providers are accepted.
    CONSTRAINT user_oauth_identities_provider_check
        CHECK (provider IN ('google', 'apple'))
);

-- Speeds up OAuth identity lookups by user.
CREATE INDEX idx_user_oauth_identities_user_id
    ON user_oauth_identities (user_id);