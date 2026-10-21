-- Prepared snapshots from Phase 7.2 remain private until fresh confirmation.
ALTER TABLE business_public_profiles ADD COLUMN public_confirmed BOOLEAN NOT NULL DEFAULT false;
