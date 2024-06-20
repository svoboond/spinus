-- name: ListMainMeterBillingSubMeters :many
SELECT	sub_meter.id,
	sub_meter.subid,
	sub_meter.meter_id,
	spinus_user.email,
	energy_consumption,
	consumed_energy_price,
	service_price,
	advance_price,
	total_price
FROM	sub_meter_billing
JOIN	sub_meter
	on sub_meter_billing.fk_sub_meter = sub_meter.id
JOIN	spinus_user
	ON sub_meter.fk_user = spinus_user.id
WHERE	fk_main_billing = $1
ORDER BY subid;

-- name: GetPreviousSubMeterBillingAdvancePrices :many
SELECT	sub_meter.subid,
	sub_meter_billing.advance_price
FROM (
	SELECT		id
	FROM		main_meter_billing
	WHERE		fk_main_meter = sqlc.arg(main_meter_id)
	ORDER BY 	subid DESC
	LIMIT 1
) previous_main_meter_billing
JOIN	sub_meter_billing
	ON previous_main_meter_billing.id = sub_meter_billing.fk_main_billing
JOIN	sub_meter
	ON sub_meter_billing.fk_sub_meter = sub_meter.id;
