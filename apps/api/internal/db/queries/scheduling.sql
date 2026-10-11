-- name: ListLocationWeeklyHours :many
SELECT weekday,start_minute,end_minute FROM location_weekly_hours
WHERE business_id=$1 AND location_id=$2 ORDER BY weekday,start_minute;

-- name: DeleteLocationWeeklyHours :exec
DELETE FROM location_weekly_hours WHERE business_id=$1 AND location_id=$2;

-- name: CreateLocationWeeklyHours :exec
INSERT INTO location_weekly_hours(business_id,location_id,weekday,start_minute,end_minute) VALUES($1,$2,$3,$4,$5);

-- name: ListLocationClosedDates :many
SELECT closed_date FROM location_closed_dates WHERE business_id=$1 AND location_id=$2 ORDER BY closed_date;

-- name: DeleteLocationClosedDates :exec
DELETE FROM location_closed_dates WHERE business_id=$1 AND location_id=$2;

-- name: CreateLocationClosedDate :exec
INSERT INTO location_closed_dates(business_id,location_id,closed_date) VALUES($1,$2,$3);

-- name: GetServiceSchedulingRules :one
SELECT * FROM service_scheduling_rules WHERE business_id=$1 AND service_id=$2;

-- name: PutServiceSchedulingRules :exec
INSERT INTO service_scheduling_rules(business_id,service_id,buffer_before_minutes,buffer_after_minutes,minimum_lead_minutes)
VALUES($1,$2,$3,$4,$5)
ON CONFLICT(business_id,service_id) DO UPDATE SET
 buffer_before_minutes=EXCLUDED.buffer_before_minutes,
 buffer_after_minutes=EXCLUDED.buffer_after_minutes,
 minimum_lead_minutes=EXCLUDED.minimum_lead_minutes,
 updated_at=NOW();
