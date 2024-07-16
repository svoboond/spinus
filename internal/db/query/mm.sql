-- name: GetMm :one
SELECT	mm.*, spinus_user.email
FROM	mm
JOIN	spinus_user
	ON mm.fk_user = spinus_user.id
WHERE	mm.id = $1
LIMIT	1;

-- name: ListUserMms :many
SELECT	*
FROM	mm
WHERE	fk_user = $1
ORDER BY id;

-- name: CreateMm :one
INSERT INTO mm (
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
