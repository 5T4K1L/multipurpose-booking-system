DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM bookings WHERE customer_id IS NOT NULL)
    OR EXISTS (SELECT 1 FROM businesses WHERE is_published) THEN
  RAISE EXCEPTION 'cannot discard customer ownership or publication state';
 END IF;
END $$;
DROP TRIGGER bookings_customer_guard ON bookings;
DROP FUNCTION guard_booking_customer();
ALTER TABLE bookings DROP COLUMN customer_id;
ALTER TABLE businesses DROP COLUMN is_published;
