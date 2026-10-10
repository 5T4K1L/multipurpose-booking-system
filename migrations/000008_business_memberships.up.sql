CREATE TABLE businesses (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 100),
 timezone TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE business_memberships (
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
 user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 role TEXT NOT NULL CHECK (role IN ('BUSINESS_OWNER','BUSINESS_ADMIN','MANAGER','STAFF')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (business_id,user_id)
);
CREATE INDEX business_memberships_user_idx ON business_memberships(user_id,business_id);
