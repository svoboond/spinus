-- name: GetSubMeterReadings :many
SELECT
	sub_meter.id,
	sub_meter_reading.reading_value,
	sub_meter_reading.reading_date
FROM sub_meter
LEFT JOIN sub_meter_reading
	ON sub_meter.id = sub_meter_reading.fk_sub_meter
WHERE fk_main_meter = $1
ORDER BY
	sub_meter.id ASC,
	reading_date ASC;

-- name: CreateMainMeterBillingPeriod :one
INSERT INTO main_meter_billing_period (
	fk_main_billing,
	subid,
	begin_date,
	end_date,
	max_day_diff,
	begin_reading_value,
	end_reading_value,
	consumed_energy_price,
	service_price
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3, $4, $5, $6, $7, $8
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
