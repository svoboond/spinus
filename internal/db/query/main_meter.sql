-- name: GetMainMeter :one
SELECT main_meter.*, spinus_user.email
FROM main_meter
JOIN spinus_user
	ON main_meter.fk_user = spinus_user.id
WHERE main_meter.id = $1
LIMIT 1;

-- name: ListUserMainMeters :many
SELECT * FROM main_meter
WHERE fk_user = $1
ORDER BY id;

-- name: CreateMainMeter :one
INSERT INTO main_meter (
	meter_id,
	energy,
	address,
	currency_code,
	fk_user
) VALUES (
	TRIM(sqlc.arg(meter_id)),
	sqlc.arg(energy),
	TRIM(sqlc.arg(address)),
	UPPER(TRIM(sqlc.arg(currency_code))),
	sqlc.arg(fk_user)
)
RETURNING *;

-- name: UpdateMainMeter :exec
UPDATE main_meter set
	meter_id = $2,
	energy = $3,
	address = $4,
	currency_code = $5,
	fk_user = $6
WHERE id = $1;

-- name: DeleteMainMeter :exec
DELETE FROM main_meter
WHERE id = $1;
