-- name: ListMainMeterBillingSubMeters :many
SELECT	sub_meter.id,
	sub_meter.subid,
	sub_meter.meter_id,
	spinus_user.email,
	energy_consumption,
	consumed_energy_price,
	service_price,
	advance_price,
	from_financial_balance,
	to_pay,
	status
FROM	sub_meter_billing
JOIN	sub_meter
	on sub_meter_billing.fk_sub_meter = sub_meter.id
JOIN	spinus_user
	ON sub_meter.fk_user = spinus_user.id
WHERE	fk_main_billing = $1
ORDER BY subid;
