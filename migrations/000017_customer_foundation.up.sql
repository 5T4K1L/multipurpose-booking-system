-- Existing operator records deliberately have no customer owner.
ALTER TABLE bookings ADD COLUMN customer_id BIGINT REFERENCES users(id) ON DELETE RESTRICT;
CREATE INDEX bookings_customer_history ON bookings(customer_id,id) WHERE customer_id IS NOT NULL;
ALTER TABLE businesses ADD COLUMN is_published BOOLEAN NOT NULL DEFAULT false;

-- The existing transition guard makes customer_id immutable on UPDATE.
-- Replacements must retain ownership, including NULL for internal bookings.
CREATE FUNCTION guard_booking_customer() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_customer BIGINT;
BEGIN
 IF NEW.supersedes_booking_id IS NOT NULL THEN
  SELECT customer_id INTO parent_customer FROM bookings
   WHERE business_id=NEW.business_id AND id=NEW.supersedes_booking_id;
  IF NOT FOUND OR NEW.customer_id IS DISTINCT FROM parent_customer THEN
   RAISE EXCEPTION 'replacement customer mismatch' USING ERRCODE='23514', CONSTRAINT='bookings_customer_chain';
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER bookings_customer_guard BEFORE INSERT ON bookings FOR EACH ROW EXECUTE FUNCTION guard_booking_customer();
