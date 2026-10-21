-- Empty on upgrade: Phase 7.1 flags alone never expose contacts.
CREATE TABLE business_public_profiles (
 business_id UUID PRIMARY KEY REFERENCES businesses(id) ON DELETE CASCADE,
 profile JSONB NOT NULL CHECK (jsonb_typeof(profile)='object'),
 published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
