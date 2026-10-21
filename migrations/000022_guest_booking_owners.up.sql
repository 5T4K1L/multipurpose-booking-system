-- Guest principals have no credentials, OAuth identities or sign-in sessions.
CREATE TABLE guest_visitors (user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, token_hash text UNIQUE NOT NULL CHECK(length(token_hash)=64), created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE guest_booking_contacts (booking_id uuid PRIMARY KEY REFERENCES bookings(id) ON DELETE CASCADE, name text NOT NULL, phone text NOT NULL, email text NOT NULL DEFAULT '', notes text NOT NULL DEFAULT '', hidden boolean NOT NULL DEFAULT false);
CREATE UNIQUE INDEX one_owned_business_per_user ON business_memberships(user_id) WHERE role='BUSINESS_OWNER';
CREATE TABLE business_images (business_id uuid PRIMARY KEY REFERENCES businesses(id) ON DELETE CASCADE, content bytea NOT NULL DEFAULT '', content_type text NOT NULL DEFAULT 'image/jpeg', published_content bytea NOT NULL DEFAULT '', published_content_type text NOT NULL DEFAULT 'image/jpeg');

CREATE FUNCTION reject_guest_signin() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF EXISTS(SELECT 1 FROM guest_visitors WHERE user_id=NEW.user_id) THEN RAISE EXCEPTION 'Guest principals cannot sign in or join businesses' USING ERRCODE='23514'; END IF;
 RETURN NEW; END $$;
CREATE TRIGGER guest_no_credentials BEFORE INSERT OR UPDATE ON user_credentials FOR EACH ROW EXECUTE FUNCTION reject_guest_signin();
CREATE TRIGGER guest_no_oauth BEFORE INSERT OR UPDATE ON user_oauth_identities FOR EACH ROW EXECUTE FUNCTION reject_guest_signin();
CREATE TRIGGER guest_no_sessions BEFORE INSERT OR UPDATE ON user_sessions FOR EACH ROW EXECUTE FUNCTION reject_guest_signin();
CREATE TRIGGER guest_no_memberships BEFORE INSERT OR UPDATE ON business_memberships FOR EACH ROW EXECUTE FUNCTION reject_guest_signin();
