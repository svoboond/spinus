-- name: GetSmRdgs :many
WITH selected_sm AS (
	SELECT sm.id
	FROM sm
	WHERE fk_mm = sqlc.arg(fk_mm)
)
SELECT later_rdg.sm_id, sm_rdg.rdg_val, sm_rdg.rdg_date
FROM (
	SELECT selected_sm.id AS sm_id, min(sm_rdg.rdg_date) AS rdg_date
	FROM selected_sm
	JOIN sm_rdg
	ON selected_sm.id = sm_rdg.fk_sm
	WHERE sm_rdg.rdg_date > sqlc.arg(date_max)
	GROUP BY selected_sm.id
) later_rdg
LEFT JOIN sm_rdg
	ON later_rdg.sm_id = sm_rdg.fk_sm AND later_rdg.rdg_date = sm_rdg.rdg_date
UNION
SELECT selected_sm.id AS sm_id, sm_rdg.rdg_val, sm_rdg.rdg_date
FROM selected_sm
JOIN sm_rdg
	ON selected_sm.id = sm_rdg.fk_sm
WHERE sm_rdg.rdg_date BETWEEN sqlc.arg(date_min) AND sqlc.arg(date_max)
UNION
SELECT selected_sm.id AS sm_id, sm_rdg.rdg_val, earlier_rdg.rdg_date
FROM selected_sm
LEFT JOIN (
	SELECT selected_sm.id AS sm_id, max(sm_rdg.rdg_date) AS rdg_date
	FROM selected_sm
	LEFT JOIN sm_rdg
		ON	selected_sm.id = sm_rdg.fk_sm
	WHERE sm_rdg.rdg_date < sqlc.arg(date_min)
	GROUP BY selected_sm.id
) earlier_rdg
	ON selected_sm.id = earlier_rdg.sm_id
LEFT JOIN sm_rdg
	ON earlier_rdg.sm_id = sm_rdg.fk_sm AND earlier_rdg.rdg_date = sm_rdg.rdg_date
ORDER BY rdg_date DESC NULLS LAST;

-- name: CreateMmBill :one
INSERT INTO mm_bill (
	id,
	fk_mm,
	max_day_diff,
	begin_date,
	end_date,
	energy_consum,
	consum_energy_price,
	service_price,
	advance_price,
	from_fin_balance,
	to_pay,
	status
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: CreateMmBillPeriod :one
INSERT INTO mm_bill_period (
	id,
	fk_mm_bill,
	begin_date,
	end_date,
	begin_rdg_val,
	end_rdg_val,
	energy_consum,
	consum_energy_price,
	service_price,
	advance_price,
	total_price
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: CreateSmBill :one
INSERT INTO sm_bill (
	id,
	fk_sm,
	fk_mm_bill,
	energy_consum,
	consum_energy_price,
	service_price,
	advance_price,
	from_fin_balance,
	to_pay,
	status
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: CreateSmBillPeriod :one
INSERT INTO sm_bill_period (
	id,
	fk_sm_bill,
	fk_mm_bill_period,
	energy_consum,
	consum_energy_price,
	service_price,
	advance_price,
	total_price
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;
