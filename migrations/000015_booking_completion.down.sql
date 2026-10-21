-- Never discard completion history during rollback.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM bookings WHERE status='completed') OR EXISTS (SELECT 1 FROM booking_changes WHERE action='complete') THEN
 RAISE EXCEPTION 'cannot downgrade with completion history'; END IF;
END $$;
ALTER TABLE bookings DROP CONSTRAINT bookings_status_check;
ALTER TABLE bookings ADD CONSTRAINT bookings_status_check CHECK (status IN ('confirmed','cancelled','rescheduled'));
ALTER TABLE bookings DROP CONSTRAINT bookings_closed_state;
ALTER TABLE bookings ADD CONSTRAINT bookings_closed_state CHECK (
 (status='confirmed' AND closed_at IS NULL AND closed_by IS NULL) OR
 (status IN ('cancelled','rescheduled') AND closed_at IS NOT NULL AND isfinite(closed_at) AND closed_by IS NOT NULL));
ALTER TABLE booking_changes DROP CONSTRAINT booking_changes_action_check;
ALTER TABLE booking_changes ADD CONSTRAINT booking_changes_action_check CHECK (action IN ('cancel','reschedule'));
CREATE OR REPLACE FUNCTION guard_booking_transition() RETURNS trigger LANGUAGE plpgsql AS $$
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
