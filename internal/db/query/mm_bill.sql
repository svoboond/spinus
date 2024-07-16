-- name: ListMmBills :many
SELECT *
FROM mm_bill
WHERE fk_mm = $1
ORDER BY subid DESC;

-- name: GetMmBill :one
SELECT mm_bill.*
FROM mm_bill
JOIN mm
	ON mm_bill.fk_mm = mm.id
WHERE fk_mm = $1 and subid = $2
LIMIT 1;

-- name: ListMmBillSms :many
SELECT
	sm.id,
	sm.subid,
	sm.meter_id,
	spinus_user.email,
	energy_consum,
	consum_energy_price,
	service_price,
	advance_price,
	from_fin_balance,
	to_pay,
	status
FROM sm_bill
JOIN sm
	ON sm_bill.fk_sm = sm.id
JOIN spinus_user
	ON sm.fk_user = spinus_user.id
WHERE fk_mm_bill = $1
ORDER BY subid;
