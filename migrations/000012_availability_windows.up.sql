CREATE TABLE staff_availability_windows (
 business_id UUID NOT NULL,
 staff_id UUID NOT NULL,
 starts_at TIMESTAMPTZ NOT NULL,
 ends_at TIMESTAMPTZ NOT NULL,
 CHECK (isfinite(starts_at) AND isfinite(ends_at) AND ends_at > starts_at AND ends_at - starts_at <= INTERVAL '7 days'),
 PRIMARY KEY (business_id,staff_id,starts_at),
 FOREIGN KEY (business_id,staff_id) REFERENCES staff_profiles(business_id,id) ON DELETE RESTRICT
);
CREATE TABLE resource_availability_windows (
 business_id UUID NOT NULL,
 resource_id UUID NOT NULL,
 starts_at TIMESTAMPTZ NOT NULL,
 ends_at TIMESTAMPTZ NOT NULL,
 CHECK (isfinite(starts_at) AND isfinite(ends_at) AND ends_at > starts_at AND ends_at - starts_at <= INTERVAL '7 days'),
 PRIMARY KEY (business_id,resource_id,starts_at),
 FOREIGN KEY (business_id,resource_id) REFERENCES resources(business_id,id) ON DELETE RESTRICT
);
