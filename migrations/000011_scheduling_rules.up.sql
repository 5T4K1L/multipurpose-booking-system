CREATE TABLE location_weekly_hours (
 business_id UUID NOT NULL,
 location_id UUID NOT NULL,
 weekday INTEGER NOT NULL CHECK (weekday BETWEEN 0 AND 6),
 start_minute INTEGER NOT NULL CHECK (start_minute BETWEEN 0 AND 1439),
 end_minute INTEGER NOT NULL CHECK (end_minute BETWEEN 1 AND 1440),
 CHECK (start_minute < end_minute),
 PRIMARY KEY (business_id,location_id,weekday,start_minute),
 FOREIGN KEY (business_id,location_id) REFERENCES locations(business_id,id) ON DELETE RESTRICT
);
CREATE TABLE location_closed_dates (
 business_id UUID NOT NULL,
 location_id UUID NOT NULL,
 closed_date DATE NOT NULL CHECK (closed_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),
 PRIMARY KEY (business_id,location_id,closed_date),
 FOREIGN KEY (business_id,location_id) REFERENCES locations(business_id,id) ON DELETE RESTRICT
);
CREATE TABLE service_scheduling_rules (
 business_id UUID NOT NULL,
 service_id UUID NOT NULL,
 buffer_before_minutes INTEGER NOT NULL CHECK (buffer_before_minutes BETWEEN 0 AND 1440),
 buffer_after_minutes INTEGER NOT NULL CHECK (buffer_after_minutes BETWEEN 0 AND 1440),
 minimum_lead_minutes INTEGER NOT NULL CHECK (minimum_lead_minutes BETWEEN 0 AND 525600),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (business_id,service_id),
 FOREIGN KEY (business_id,service_id) REFERENCES services(business_id,id) ON DELETE RESTRICT
);
