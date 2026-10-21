-- name: GetServiceBookingOptions :one
SELECT (coalesce(o.enabled,false) AND NOT s.all_day)::boolean AS enabled,
 coalesce(o.minimum_minutes,s.duration_minutes)::integer AS minimum_minutes,
 coalesce(o.maximum_minutes,s.duration_minutes)::integer AS maximum_minutes
FROM services s LEFT JOIN service_booking_options o ON o.business_id=s.business_id AND o.service_id=s.id
WHERE s.business_id=$1 AND s.id=$2 AND s.status='active';

-- name: PutServiceBookingOptions :exec
INSERT INTO service_booking_options(business_id,service_id,enabled,minimum_minutes,maximum_minutes)
VALUES($1,$2,$3,$4,$5)
ON CONFLICT(business_id,service_id) DO UPDATE SET enabled=excluded.enabled,minimum_minutes=excluded.minimum_minutes,maximum_minutes=excluded.maximum_minutes;
