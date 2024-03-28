-- name: GetSubMeterReadings :many
SELECT
	later_reading.id,
	sub_meter_reading.reading_value,
	later_reading.reading_date
FROM (
	SELECT		sub_meter.id,
			min(sub_meter_reading.reading_date) AS reading_date
	FROM		sub_meter
	JOIN		sub_meter_reading
	ON		sub_meter.id = sub_meter_reading.fk_sub_meter
	WHERE		sub_meter.fk_main_meter = $1 AND
			sub_meter_reading.reading_date > sqlc.arg(date_max)
	GROUP BY	sub_meter.id
) later_reading
JOIN	sub_meter_reading
ON	later_reading.id = sub_meter_reading.id AND
	later_reading.reading_date = sub_meter_reading.reading_date
UNION
SELECT	sub_meter.id,
	sub_meter_reading.reading_value,
	sub_meter_reading.reading_date
FROM	sub_meter
JOIN	sub_meter_reading
ON	sub_meter.id = sub_meter_reading.fk_sub_meter
WHERE	sub_meter.fk_main_meter = $1 AND
	sub_meter_reading.reading_date BETWEEN sqlc.arg(date_min) AND sqlc.arg(date_max)
UNION
SELECT		earlier_reading.id,
		sub_meter_reading.reading_value,
		earlier_reading.reading_date
FROM (
	SELECT		sub_meter.id,
			max(sub_meter_reading.reading_date) AS reading_date
	FROM		sub_meter
	JOIN		sub_meter_reading
	ON		sub_meter.id = sub_meter_reading.fk_sub_meter
	WHERE		sub_meter.fk_main_meter = $1 AND reading_date < sqlc.arg(date_min)
	GROUP BY	sub_meter.id
) earlier_reading
RIGHT JOIN	sub_meter_reading
ON		earlier_reading.id = sub_meter_reading.id AND
		earlier_reading.reading_date = sub_meter_reading.reading_date
ORDER BY 	reading_date DESC NULLS LAST;

-- name: CreateMainMeterBillingPeriod :one
INSERT INTO main_meter_billing_period (
	fk_main_billing,
	subid,
	begin_date,
	end_date,
	begin_reading_value,
	end_reading_value,
	consumed_energy_price,
	service_price
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3, $4, $5, $6, $7
	FROM main_meter_billing_period
	WHERE fk_main_billing = $1
RETURNING *;

-- name: CreateSubMeterBillingPeriod :one
INSERT INTO sub_meter_billing_period (
	fk_sub_billing,
	fk_main_billing_period,
	energy_consumption,
	consumed_energy_payment,
	service_payment,
	advance_payment,
	total_payment
) VALUES (
	$1, $2, $3, $4, $5, $6, $7
)
RETURNING *;
