CREATE TABLE business_settings (
 business_id UUID PRIMARY KEY REFERENCES businesses(id) ON DELETE CASCADE,
 tagline TEXT NOT NULL DEFAULT '' CHECK (char_length(tagline)<=120),
 description TEXT NOT NULL DEFAULT '' CHECK (char_length(description)<=2000),
 contact_email TEXT NOT NULL DEFAULT '' CHECK (char_length(contact_email)<=254),
 contact_phone TEXT NOT NULL DEFAULT '' CHECK (char_length(contact_phone)<=40),
 website TEXT NOT NULL DEFAULT '' CHECK (char_length(website)<=500),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
