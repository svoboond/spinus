-- name: ListSmBills :many
SELECT sm_bill.*
FROM sm_bill
JOIN sm
	ON sm_bill.fk_sm = sm.id
WHERE fk_mm = $1
ORDER BY sm_bill.subid DESC;
