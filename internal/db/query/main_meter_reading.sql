-- name: ListMainMeterReadings :many
SELECT * FROM main_meter_reading
WHERE fk_main_meter = $1
ORDER BY reading_date DESC;

-- name: CreateMainMeterReading :one
INSERT INTO main_meter_reading (
	fk_main_meter, reading_value, reading_date
) VALUES (
	$1, $2, $3
)
RETURNING *;
