-- Catalog records use restrictive deletion so future booking history cannot be
-- removed by deleting a business or location. Lifecycle operations come later.
CREATE TABLE locations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 100),
 timezone TEXT NOT NULL CHECK (char_length(timezone) BETWEEN 1 AND 100),
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (business_id,id)
);
CREATE TABLE services (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 100),
 duration_minutes INTEGER NOT NULL CHECK (duration_minutes BETWEEN 1 AND 10080),
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (business_id,id)
);
-- A bookable staff profile is not an authentication identity or membership grant.
CREATE TABLE staff_profiles (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 100),
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (business_id,id)
);
CREATE TABLE resources (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 location_id UUID NOT NULL,
 name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 100),
 capacity INTEGER NOT NULL CHECK (capacity > 0),
 status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (business_id,id),
 FOREIGN KEY (business_id,location_id) REFERENCES locations(business_id,id) ON DELETE RESTRICT
);
CREATE INDEX resources_business_location_idx ON resources(business_id,location_id);
