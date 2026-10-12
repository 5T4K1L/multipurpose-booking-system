-- Refuse a destructive downgrade once lifecycle history exists.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM bookings WHERE status <> 'confirmed' OR supersedes_booking_id IS NOT NULL)
 OR EXISTS (SELECT 1 FROM booking_changes) THEN
   RAISE EXCEPTION 'cannot downgrade booking lifecycle with recorded history';
 END IF;
END $$;
DROP TABLE booking_changes;
DROP TRIGGER bookings_replacement_required ON bookings;
DROP FUNCTION require_booking_replacement();
DROP TRIGGER bookings_transition ON bookings;
DROP FUNCTION guard_booking_transition();
ALTER TABLE bookings DROP CONSTRAINT bookings_closed_state;
ALTER TABLE bookings DROP CONSTRAINT bookings_supersedes_fk;
ALTER TABLE bookings DROP CONSTRAINT bookings_one_replacement;
ALTER TABLE bookings DROP CONSTRAINT bookings_business_id_id_key;
ALTER TABLE bookings DROP COLUMN supersedes_booking_id, DROP COLUMN closed_at, DROP COLUMN closed_by;
ALTER TABLE bookings DROP CONSTRAINT bookings_status_check;
ALTER TABLE bookings ADD CONSTRAINT bookings_status_check CHECK (status='confirmed');
DROP INDEX bookings_staff_occupancy;
DROP INDEX bookings_resource_occupancy;
CREATE INDEX bookings_staff_occupancy ON bookings(business_id,staff_id,busy_starts_at,busy_ends_at) WHERE staff_id IS NOT NULL;
CREATE INDEX bookings_resource_occupancy ON bookings(business_id,resource_id,busy_starts_at,busy_ends_at) WHERE resource_id IS NOT NULL;
CREATE OR REPLACE FUNCTION guard_booking_insert() RETURNS trigger LANGUAGE plpgsql AS $$
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
CREATE FUNCTION reject_booking_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'booking lifecycle changes are not implemented' USING ERRCODE='23514'; END $$;
CREATE TRIGGER bookings_immutable BEFORE UPDATE ON bookings FOR EACH ROW EXECUTE FUNCTION reject_booking_update();
