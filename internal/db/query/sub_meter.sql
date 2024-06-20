-- name: GetSubMeter :one
SELECT
	sub_meter.id,
	sub_meter.fk_main_meter AS main_meter_id,
	sub_meter.subid,
	sub_meter.meter_id AS sub_meter_id,
	sub_meter.financial_balance,
	sub_meter.fk_user AS sub_user_id,
	sub_user.email AS sub_user_email,
	main_meter.address,
	main_meter.fk_user AS main_user_id,
	main_user.email AS main_user_email
FROM sub_meter
JOIN main_meter
	ON sub_meter.fk_main_meter = main_meter.id
JOIN spinus_user AS sub_user
	ON sub_meter.fk_user = sub_user.id
JOIN spinus_user AS main_user
	ON main_meter.fk_user = main_user.id
WHERE fk_main_meter = $1 AND subid = $2
LIMIT 1;

-- name: ListSubMeters :many
SELECT sub_meter.id, subid, meter_id, financial_balance, email
FROM sub_meter
JOIN spinus_user
	ON sub_meter.fk_user = spinus_user.id
WHERE fk_main_meter = $1
ORDER BY subid;

-- name: CreateSubMeter :one
INSERT INTO sub_meter (
	fk_main_meter, subid, meter_id, financial_balance, fk_user
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3, $4
	FROM sub_meter
	WHERE fk_main_meter = $1
RETURNING *;
