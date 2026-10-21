ALTER TABLE services ADD COLUMN all_day BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE services ADD CONSTRAINT services_all_day_duration CHECK (NOT all_day OR duration_minutes=1440);
