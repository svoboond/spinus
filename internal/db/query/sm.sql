-- name: GetSm :one
SELECT
	sm.id,
	sm.fk_mm AS mm_id,
	sm.meter_id,
	sm.fin_balance,
	sm.fk_user AS sub_user_id,
	sub_user.email AS sub_user_email,
	mm.address,
	mm.fk_user AS main_user_id,
	main_user.email AS main_user_email
FROM sm
JOIN mm
	ON sm.fk_mm = mm.id
JOIN spinus_user AS sub_user
	ON sm.fk_user = sub_user.id
JOIN spinus_user AS main_user
	ON mm.fk_user = main_user.id
WHERE sm.id = $1
LIMIT 1;

-- name: ListSms :many
SELECT sm.id, sm.created_ts, meter_id, fin_balance, email
FROM sm
JOIN spinus_user
	ON sm.fk_user = spinus_user.id
WHERE fk_mm = $1
ORDER BY sm.created_ts;

-- name: CreateSm :one
INSERT INTO sm (
	id, fk_mm, meter_id, fin_balance, fk_user
) VALUES (
	$1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateSmFinBalance :exec
UPDATE sm
SET fin_balance = $2
WHERE id = $1;
