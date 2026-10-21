CREATE TABLE business_booking_dismissals (
 business_id UUID NOT NULL,
 booking_id UUID NOT NULL,
 deleted_by BIGINT NOT NULL REFERENCES users(id),
 deleted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(business_id,booking_id),
 FOREIGN KEY(business_id,booking_id) REFERENCES bookings(business_id,id) ON DELETE CASCADE
);
