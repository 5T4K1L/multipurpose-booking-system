CREATE TABLE service_locations (
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 service_id UUID NOT NULL,
 location_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (business_id,service_id,location_id),
 FOREIGN KEY (business_id,service_id) REFERENCES services(business_id,id) ON DELETE RESTRICT,
 FOREIGN KEY (business_id,location_id) REFERENCES locations(business_id,id) ON DELETE RESTRICT
);
CREATE INDEX service_locations_target_idx ON service_locations(business_id,location_id);
CREATE TABLE service_staff (
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 service_id UUID NOT NULL,
 staff_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (business_id,service_id,staff_id),
 FOREIGN KEY (business_id,service_id) REFERENCES services(business_id,id) ON DELETE RESTRICT,
 FOREIGN KEY (business_id,staff_id) REFERENCES staff_profiles(business_id,id) ON DELETE RESTRICT
);
CREATE INDEX service_staff_target_idx ON service_staff(business_id,staff_id);
CREATE TABLE service_resources (
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 service_id UUID NOT NULL,
 resource_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (business_id,service_id,resource_id),
 FOREIGN KEY (business_id,service_id) REFERENCES services(business_id,id) ON DELETE RESTRICT,
 FOREIGN KEY (business_id,resource_id) REFERENCES resources(business_id,id) ON DELETE RESTRICT
);
CREATE INDEX service_resources_target_idx ON service_resources(business_id,resource_id);
CREATE TABLE staff_locations (
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 staff_id UUID NOT NULL,
 location_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (business_id,staff_id,location_id),
 FOREIGN KEY (business_id,staff_id) REFERENCES staff_profiles(business_id,id) ON DELETE RESTRICT,
 FOREIGN KEY (business_id,location_id) REFERENCES locations(business_id,id) ON DELETE RESTRICT
);
CREATE INDEX staff_locations_target_idx ON staff_locations(business_id,location_id);
