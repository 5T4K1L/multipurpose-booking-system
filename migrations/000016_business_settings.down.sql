-- Prevent silent loss of saved customization.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM business_settings WHERE tagline<>'' OR description<>'' OR contact_email<>'' OR contact_phone<>'' OR website<>'') THEN
 RAISE EXCEPTION 'cannot downgrade with saved business customization'; END IF;
END $$;
DROP TABLE business_settings;
