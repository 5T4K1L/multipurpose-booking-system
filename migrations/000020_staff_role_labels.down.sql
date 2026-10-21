DO $$ BEGIN IF EXISTS (SELECT 1 FROM staff_profiles WHERE role_name<>'') THEN RAISE EXCEPTION 'Clear custom staff roles before downgrading'; END IF; END $$;
ALTER TABLE staff_profiles DROP COLUMN role_name;
