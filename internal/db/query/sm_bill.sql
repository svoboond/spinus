-- name: ListSmBills :many
SELECT *
FROM sm_bill
WHERE fk_sm = $1
ORDER BY created_ts DESC;

-- name: GetSmBill :one
SELECT
	sm_bill.*,
	sm.fk_user AS sub_user_id,
	mm.fk_user AS main_user_id
FROM sm_bill
JOIN sm
	ON sm_bill.fk_sm = sm.id
JOIN mm_bill
	ON sm_bill.fk_mm_bill = mm_bill.id
JOIN mm
	ON mm_bill.fk_mm = mm.id
WHERE sm_bill.id = $1
LIMIT 1;
