DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM business_public_profiles) THEN
  RAISE EXCEPTION 'cannot discard reviewed business profiles';
 END IF;
END $$;
DROP TABLE business_public_profiles;
