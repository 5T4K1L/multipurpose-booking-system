-- name: GetBusinessSettings :one
SELECT * FROM business_settings WHERE business_id=$1;

-- name: PutBusinessSettings :one
INSERT INTO business_settings(business_id,tagline,description,contact_email,contact_phone,website)
VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT(business_id) DO UPDATE SET tagline=EXCLUDED.tagline,description=EXCLUDED.description,contact_email=EXCLUDED.contact_email,contact_phone=EXCLUDED.contact_phone,website=EXCLUDED.website,updated_at=now()
RETURNING *;
