ALTER TABLE staff_profiles ADD COLUMN role_name TEXT NOT NULL DEFAULT '' CHECK (char_length(role_name)<=60);
