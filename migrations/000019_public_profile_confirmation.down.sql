DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM business_public_profiles) THEN
  RAISE EXCEPTION 'cannot remove public confirmation boundary while profiles exist';
 END IF;
END $$;
ALTER TABLE business_public_profiles DROP COLUMN public_confirmed;
