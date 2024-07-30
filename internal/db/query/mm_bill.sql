-- name: ListMmBills :many
SELECT *
FROM mm_bill
WHERE fk_mm = $1
ORDER BY created_ts DESC;

-- name: GetMmBill :one
SELECT mm_bill.*, mm.fk_user
FROM mm_bill
JOIN mm
	ON mm_bill.fk_mm = mm.id
WHERE mm_bill.id = $1
LIMIT 1;

-- name: ListMmBillPeriods :many
SELECT mm_bill_period.*
FROM mm_bill
JOIN mm_bill_period
	ON mm_bill.id = mm_bill_period.fk_mm_bill
WHERE mm_bill.id = $1
ORDER BY mm_bill_period.begin_date;

-- name: ListMmBillSms :many
SELECT
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
JOIN sm_bill
	ON mm_bill.id = sm_bill.fk_mm_bill
JOIN sm
	ON sm_bill.fk_sm = sm.id
JOIN spinus_user
	ON sm.fk_user = spinus_user.id
WHERE mm_bill.id = $1
ORDER BY sm.created_ts;
