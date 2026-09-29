ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
    CONSTRAINT users_status_check CHECK (status IN ('active', 'disabled'));
