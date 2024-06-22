-- name: GetSubMeterReadings :many
WITH	selected_sub_meter AS (
	SELECT	sub_meter.id
	FROM	sub_meter
	WHERE	fk_main_meter = sqlc.arg(fk_main_meter)
)
SELECT		later_reading.sub_meter_id,
		sub_meter_reading.reading_value,
		sub_meter_reading.reading_date
FROM (
	SELECT		selected_sub_meter.id AS sub_meter_id,
			min(sub_meter_reading.reading_date) AS reading_date
	FROM		selected_sub_meter
	JOIN		sub_meter_reading
	ON		selected_sub_meter.id = sub_meter_reading.fk_sub_meter
	WHERE		sub_meter_reading.reading_date > sqlc.arg(date_max)
	GROUP BY	selected_sub_meter.id
) later_reading
LEFT JOIN	sub_meter_reading
ON		later_reading.sub_meter_id = sub_meter_reading.fk_sub_meter AND
		later_reading.reading_date = sub_meter_reading.reading_date
UNION
SELECT	selected_sub_meter.id AS sub_meter_id,
	sub_meter_reading.reading_value,
	sub_meter_reading.reading_date
FROM	selected_sub_meter
JOIN	sub_meter_reading
ON	selected_sub_meter.id = sub_meter_reading.fk_sub_meter
WHERE	sub_meter_reading.reading_date BETWEEN
	sqlc.arg(date_min) AND sqlc.arg(date_max)
UNION
SELECT		selected_sub_meter.id AS sub_meter_id,
		sub_meter_reading.reading_value,
		earlier_reading.reading_date
FROM 		selected_sub_meter
LEFT JOIN (
	SELECT		selected_sub_meter.id AS sub_meter_id,
			max(sub_meter_reading.reading_date) AS reading_date
	FROM		selected_sub_meter
	LEFT JOIN	sub_meter_reading
	ON		selected_sub_meter.id = sub_meter_reading.fk_sub_meter
	WHERE		sub_meter_reading.reading_date < sqlc.arg(date_min)
	GROUP BY	selected_sub_meter.id
) earlier_reading
ON		selected_sub_meter.id = earlier_reading.sub_meter_id
LEFT JOIN	sub_meter_reading
ON		earlier_reading.sub_meter_id = sub_meter_reading.fk_sub_meter AND
		earlier_reading.reading_date = sub_meter_reading.reading_date
ORDER BY 	reading_date DESC NULLS LAST;

-- name: CreateMainMeterBilling :one
INSERT INTO main_meter_billing (
	fk_main_meter,
	subid,
	max_day_diff,
	begin_date,
	end_date,
	energy_consumption,
	consumed_energy_price,
	service_price,
	advance_price,
	from_financial_balance,
	to_pay,
	status
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
	FROM main_meter_billing
	WHERE fk_main_meter = $1
RETURNING *;

-- name: CreateMainMeterBillingPeriod :one
INSERT INTO main_meter_billing_period (
	fk_main_billing,
	subid,
	begin_date,
	end_date,
	begin_reading_value,
	end_reading_value,
	energy_consumption,
	consumed_energy_price,
	service_price,
	advance_price,
	total_price
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3, $4, $5, $6, $7, $8, $9, $10
	FROM main_meter_billing_period
	WHERE fk_main_billing = $1
RETURNING *;

-- name: CreateSubMeterBilling :one
INSERT INTO sub_meter_billing (
	fk_sub_meter,
	fk_main_billing,
	subid,
	energy_consumption,
	consumed_energy_price,
	service_price,
	advance_price,
	from_financial_balance,
	to_pay,
	status
) SELECT $1, $2, COALESCE(MAX(subid), 0) + 1, $3, $4, $5, $6, $7, $8, $9
	FROM sub_meter_billing
	WHERE fk_sub_meter = $1
RETURNING *;

-- name: CreateSubMeterBillingPeriod :one
INSERT INTO sub_meter_billing_period (
	fk_sub_billing,
	fk_main_billing_period,
	energy_consumption,
	consumed_energy_price,
	service_price,
	advance_price,
	total_price
) VALUES (
	$1, $2, $3, $4, $5, $6, $7
)
RETURNING *;
