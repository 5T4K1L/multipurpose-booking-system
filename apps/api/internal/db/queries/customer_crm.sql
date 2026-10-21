-- name: ListBusinessCustomers :many
SELECT u.id,coalesce(max(nullif(c.name,'')),u.username)::text AS username,CASE WHEN bool_or(g.user_id IS NOT NULL) THEN coalesce(max(nullif(c.email,'')),'') ELSE u.email END::text AS email,count(*)::bigint AS booking_records,
 count(*) FILTER (WHERE b.status='confirmed')::bigint AS confirmed_bookings,
 max(b.starts_at)::timestamptz AS latest_booking
FROM bookings b JOIN users u ON u.id=b.customer_id
LEFT JOIN guest_booking_contacts c ON c.booking_id=b.id
LEFT JOIN guest_visitors g ON g.user_id=u.id
WHERE b.business_id=sqlc.arg(business_id) AND u.id>sqlc.arg(after_user)::bigint
 AND (sqlc.arg(search)::text='' OR strpos(lower(coalesce(c.name,u.username)),lower(sqlc.arg(search)))>0 OR strpos(lower(coalesce(c.email,u.email)),lower(sqlc.arg(search)))>0)
GROUP BY u.id,u.username,u.email
ORDER BY u.id LIMIT sqlc.arg(page_size)::integer;
