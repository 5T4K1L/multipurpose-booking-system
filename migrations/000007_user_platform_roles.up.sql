-- Explicit platform grants only. Business roles require Phase 4 memberships.
-- CUSTOMER is the baseline for active local accounts, not a privileged grant.
CREATE TABLE user_platform_roles (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('PLATFORM_ADMIN', 'PLATFORM_OPERATOR')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role)
);
