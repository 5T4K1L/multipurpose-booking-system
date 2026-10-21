DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM service_booking_options WHERE enabled) THEN
  RAISE EXCEPTION 'Disable custom booking ranges before downgrading';
 END IF;
END $$;
DROP TABLE service_booking_options;
