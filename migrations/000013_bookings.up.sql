-- Booking records are immutable in this milestone; lifecycle transitions come later.
CREATE TABLE bookings (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE RESTRICT,
 service_id UUID NOT NULL,
 location_id UUID NOT NULL,
 staff_id UUID,
 resource_id UUID,
 created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 128),
 request_hash TEXT NOT NULL CHECK (length(request_hash) = 64),
 starts_at TIMESTAMPTZ NOT NULL,
 ends_at TIMESTAMPTZ NOT NULL,
 busy_starts_at TIMESTAMPTZ NOT NULL,
 busy_ends_at TIMESTAMPTZ NOT NULL,
 status TEXT NOT NULL DEFAULT 'confirmed' CHECK (status = 'confirmed'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE (business_id,created_by,idempotency_key),
 FOREIGN KEY (business_id,service_id) REFERENCES services(business_id,id) ON DELETE RESTRICT,
 FOREIGN KEY (business_id,location_id) REFERENCES locations(business_id,id) ON DELETE RESTRICT,
 FOREIGN KEY (business_id,staff_id) REFERENCES staff_profiles(business_id,id) ON DELETE RESTRICT,
 FOREIGN KEY (business_id,resource_id) REFERENCES resources(business_id,id) ON DELETE RESTRICT,
 CHECK (isfinite(busy_starts_at) AND isfinite(busy_ends_at)
   AND busy_starts_at <= starts_at AND starts_at < ends_at AND ends_at <= busy_ends_at
   AND ends_at - starts_at <= INTERVAL '7 days'
   AND starts_at - busy_starts_at <= INTERVAL '1 day'
   AND busy_ends_at - ends_at <= INTERVAL '1 day')
);
CREATE INDEX bookings_staff_occupancy ON bookings(business_id,staff_id,busy_starts_at,busy_ends_at) WHERE staff_id IS NOT NULL;
CREATE INDEX bookings_resource_occupancy ON bookings(business_id,resource_id,busy_starts_at,busy_ends_at) WHERE resource_id IS NOT NULL;

CREATE FUNCTION guard_booking_insert() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE resource_capacity INTEGER;
BEGIN
 -- Updating the tenant row serializes even direct SQL writers. Unlike a bare row
 -- lock, the version change also prevents stale repeatable-read snapshots from
 -- admitting conflicts (PostgreSQL raises a serialization failure).
 UPDATE businesses SET name = name WHERE id = NEW.business_id;
 IF NEW.staff_id IS NOT NULL AND EXISTS (
   SELECT 1 FROM bookings WHERE business_id=NEW.business_id AND staff_id=NEW.staff_id
   AND busy_starts_at < NEW.busy_ends_at AND busy_ends_at > NEW.busy_starts_at
 ) THEN RAISE EXCEPTION 'staff occupied' USING ERRCODE='23P01', CONSTRAINT='bookings_staff_conflict'; END IF;
 IF NEW.resource_id IS NOT NULL THEN
   SELECT capacity INTO resource_capacity FROM resources
     WHERE business_id=NEW.business_id AND id=NEW.resource_id AND location_id=NEW.location_id;
   IF NOT FOUND THEN RAISE EXCEPTION 'invalid resource location' USING ERRCODE='23514', CONSTRAINT='bookings_resource_location'; END IF;
   -- Clip existing occupancy to the new interval, group simultaneous ends/starts,
   -- then sweep the deltas. Half-open intervals allow touching bookings.
   IF EXISTS (
     SELECT 1 FROM (
       SELECT sum(sum(delta)) OVER (ORDER BY instant) AS occupied FROM (
         SELECT greatest(busy_starts_at,NEW.busy_starts_at) AS instant, 1 AS delta FROM bookings
         WHERE business_id=NEW.business_id AND resource_id=NEW.resource_id
           AND busy_starts_at < NEW.busy_ends_at AND busy_ends_at > NEW.busy_starts_at
         UNION ALL
         SELECT least(busy_ends_at,NEW.busy_ends_at), -1 FROM bookings
         WHERE business_id=NEW.business_id AND resource_id=NEW.resource_id
           AND busy_starts_at < NEW.busy_ends_at AND busy_ends_at > NEW.busy_starts_at
       ) events GROUP BY instant
     ) occupancy WHERE occupied >= resource_capacity
   ) THEN RAISE EXCEPTION 'resource full' USING ERRCODE='23P01', CONSTRAINT='bookings_resource_capacity'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER bookings_insert_guard BEFORE INSERT ON bookings FOR EACH ROW EXECUTE FUNCTION guard_booking_insert();

CREATE FUNCTION reject_booking_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'booking lifecycle changes are not implemented' USING ERRCODE='23514'; END $$;
CREATE TRIGGER bookings_immutable BEFORE UPDATE ON bookings FOR EACH ROW EXECUTE FUNCTION reject_booking_update();

CREATE FUNCTION guard_booked_resource_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.capacity IS DISTINCT FROM OLD.capacity OR NEW.location_id IS DISTINCT FROM OLD.location_id OR NEW.business_id IS DISTINCT FROM OLD.business_id THEN
   UPDATE businesses SET name=name WHERE id=OLD.business_id;
   IF EXISTS (SELECT 1 FROM bookings WHERE business_id=OLD.business_id AND resource_id=OLD.id) THEN
     RAISE EXCEPTION 'booked resource changes require lifecycle rules' USING ERRCODE='23514';
   END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER booked_resource_guard BEFORE UPDATE ON resources FOR EACH ROW EXECUTE FUNCTION guard_booked_resource_update();
