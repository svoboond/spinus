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
WHERE mm.id = $1 and subid = $2
LIMIT 1;

-- name: ListMmBillPeriods :many
SELECT mm_bill_period.*
FROM mm_bill
JOIN mm
	ON mm_bill.fk_mm = mm.id
JOIN mm_bill_period
	ON mm_bill.id = mm_bill_period.fk_mm_bill
WHERE mm.id = $1 and mm_bill.subid = $2
ORDER BY subid DESC;

-- name: ListMmBillSms :many
SELECT
	sm.subid,
	sm.meter_id,
	spinus_user.email,
	sm_bill.energy_consum,
	sm_bill.consum_energy_price,
	sm_bill.service_price,
	sm_bill.advance_price,
	sm_bill.from_fin_balance,
	sm_bill.to_pay,
	sm_bill.status
FROM mm_bill
JOIN mm
	ON mm_bill.fk_mm = mm.id
JOIN sm_bill
	ON mm_bill.id = sm_bill.fk_mm_bill
JOIN sm
	ON sm_bill.fk_sm = sm.id
JOIN spinus_user
	ON sm.fk_user = spinus_user.id
WHERE mm.id = $1 and mm_bill.subid = $2
ORDER BY subid;
