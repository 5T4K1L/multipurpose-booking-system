CREATE TABLE service_booking_options (
 business_id UUID NOT NULL,
 service_id UUID NOT NULL,
 enabled BOOLEAN NOT NULL DEFAULT false,
 minimum_minutes INTEGER NOT NULL CHECK (minimum_minutes BETWEEN 1 AND 10080),
 maximum_minutes INTEGER NOT NULL CHECK (maximum_minutes BETWEEN minimum_minutes AND 10080),
 PRIMARY KEY (business_id,service_id),
 FOREIGN KEY (business_id,service_id) REFERENCES services(business_id,id) ON DELETE CASCADE
);
