-- name: GetSubMeter :one
SELECT * FROM sub_meter
WHERE fk_main_meter = $1 AND subid = $2
LIMIT 1;

-- name: ListSubMeters :many
SELECT subid, meter_id, email
FROM sub_meter
JOIN spinus_user
	ON sub_meter.fk_user = spinus_user.id
WHERE fk_main_meter = $1
ORDER BY subid;

-- name: CreateSubMeter :one
INSERT INTO sub_meter (
	meter_id, subid, fk_main_meter, fk_user
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3
	FROM sub_meter
	WHERE fk_main_meter = $2
RETURNING *;
