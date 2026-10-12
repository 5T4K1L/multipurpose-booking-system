-- Retain original intervals and identity; terminal records never occupy capacity.
ALTER TABLE bookings DROP CONSTRAINT bookings_status_check;
ALTER TABLE bookings ADD CONSTRAINT bookings_status_check CHECK (status IN ('confirmed','cancelled','rescheduled'));
ALTER TABLE bookings ADD COLUMN closed_at TIMESTAMPTZ;
ALTER TABLE bookings ADD COLUMN closed_by BIGINT REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE bookings ADD COLUMN supersedes_booking_id UUID;
ALTER TABLE bookings ADD CONSTRAINT bookings_business_id_id_key UNIQUE (business_id,id);
ALTER TABLE bookings ADD CONSTRAINT bookings_supersedes_fk FOREIGN KEY (business_id,supersedes_booking_id) REFERENCES bookings(business_id,id) ON DELETE RESTRICT;
ALTER TABLE bookings ADD CONSTRAINT bookings_one_replacement UNIQUE (business_id,supersedes_booking_id);
ALTER TABLE bookings ADD CONSTRAINT bookings_closed_state CHECK (
 (status='confirmed' AND closed_at IS NULL AND closed_by IS NULL) OR
 (status IN ('cancelled','rescheduled') AND closed_at IS NOT NULL AND isfinite(closed_at) AND closed_by IS NOT NULL));
DROP INDEX bookings_staff_occupancy;
DROP INDEX bookings_resource_occupancy;
CREATE INDEX bookings_staff_occupancy ON bookings(business_id,staff_id,busy_starts_at,busy_ends_at) WHERE staff_id IS NOT NULL AND status='confirmed';
CREATE INDEX bookings_resource_occupancy ON bookings(business_id,resource_id,busy_starts_at,busy_ends_at) WHERE resource_id IS NOT NULL AND status='confirmed';

CREATE TABLE booking_changes (
 business_id UUID NOT NULL,
 actor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 128),
 booking_id UUID NOT NULL,
 action TEXT NOT NULL CHECK (action IN ('cancel','reschedule')),
 request_hash TEXT NOT NULL CHECK (length(request_hash)=64),
 response JSONB NOT NULL CHECK (jsonb_typeof(response)='object'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY (business_id,actor_id,idempotency_key),
 FOREIGN KEY (business_id,booking_id) REFERENCES bookings(business_id,id) ON DELETE RESTRICT
);

CREATE OR REPLACE FUNCTION guard_booking_insert() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE resource_capacity INTEGER; parent bookings%ROWTYPE;
BEGIN
 -- Updating the tenant row serializes even direct SQL writers. Unlike a bare row
 -- lock, the version change also prevents stale repeatable-read snapshots from
 -- admitting conflicts (PostgreSQL raises a serialization failure).
 UPDATE businesses SET name = name WHERE id = NEW.business_id;
 IF NEW.status <> 'confirmed' OR NEW.closed_at IS NOT NULL OR NEW.closed_by IS NOT NULL THEN
   RAISE EXCEPTION 'new bookings must be confirmed' USING ERRCODE='23514', CONSTRAINT='bookings_lifecycle';
 END IF;
 IF NEW.supersedes_booking_id IS NOT NULL THEN
   SELECT * INTO parent FROM bookings WHERE business_id=NEW.business_id AND id=NEW.supersedes_booking_id;
   IF NOT FOUND OR parent.status <> 'rescheduled'
     OR (NEW.service_id,NEW.location_id,NEW.staff_id,NEW.resource_id) IS DISTINCT FROM
        (parent.service_id,parent.location_id,parent.staff_id,parent.resource_id) THEN
     RAISE EXCEPTION 'invalid replacement' USING ERRCODE='23514', CONSTRAINT='bookings_lifecycle';
   END IF;
 END IF;
 IF NEW.staff_id IS NOT NULL AND EXISTS (
   SELECT 1 FROM bookings WHERE status='confirmed' AND business_id=NEW.business_id AND staff_id=NEW.staff_id
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
         WHERE status='confirmed' AND business_id=NEW.business_id AND resource_id=NEW.resource_id
           AND busy_starts_at < NEW.busy_ends_at AND busy_ends_at > NEW.busy_starts_at
         UNION ALL
         SELECT least(busy_ends_at,NEW.busy_ends_at), -1 FROM bookings
         WHERE status='confirmed' AND business_id=NEW.business_id AND resource_id=NEW.resource_id
           AND busy_starts_at < NEW.busy_ends_at AND busy_ends_at > NEW.busy_starts_at
       ) events GROUP BY instant
     ) occupancy WHERE occupied >= resource_capacity
   ) THEN RAISE EXCEPTION 'resource full' USING ERRCODE='23P01', CONSTRAINT='bookings_resource_capacity'; END IF;
 END IF;
 RETURN NEW;
END $$;

DROP TRIGGER bookings_immutable ON bookings;
DROP FUNCTION reject_booking_update();
CREATE FUNCTION guard_booking_transition() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 UPDATE businesses SET name=name WHERE id=OLD.business_id;
 IF OLD.status <> 'confirmed' OR NEW.status NOT IN ('cancelled','rescheduled')
   OR clock_timestamp() >= OLD.busy_starts_at
   OR (to_jsonb(NEW)-ARRAY['status','closed_at','closed_by']) IS DISTINCT FROM
      (to_jsonb(OLD)-ARRAY['status','closed_at','closed_by']) THEN
   RAISE EXCEPTION 'invalid booking transition' USING ERRCODE='23514', CONSTRAINT='bookings_lifecycle';
 END IF;
 NEW.closed_at := clock_timestamp();
 RETURN NEW;
END $$;
CREATE TRIGGER bookings_transition BEFORE UPDATE ON bookings FOR EACH ROW EXECUTE FUNCTION guard_booking_transition();

-- A rescheduled record must have its replacement by commit, including direct SQL.
CREATE FUNCTION require_booking_replacement() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.status='rescheduled' AND NOT EXISTS (
   SELECT 1 FROM bookings WHERE business_id=NEW.business_id AND supersedes_booking_id=NEW.id
 ) THEN RAISE EXCEPTION 'reschedule requires replacement' USING ERRCODE='23514', CONSTRAINT='bookings_lifecycle'; END IF;
 RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER bookings_replacement_required AFTER UPDATE ON bookings
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION require_booking_replacement();
