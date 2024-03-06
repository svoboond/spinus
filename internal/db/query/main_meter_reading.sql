-- name: ListMainMeterReadings :many
SELECT * FROM main_meter_reading
WHERE fk_main_meter = $1
ORDER BY reading_date DESC;

-- name: CreateMainMeterReading :one
INSERT INTO main_meter_reading (
	fk_main_meter, subid, reading_value, reading_date
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3
	FROM main_meter_reading
	WHERE fk_main_meter = $1
RETURNING *;
