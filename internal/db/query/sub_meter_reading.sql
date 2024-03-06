-- name: ListSubMeterReadings :many
SELECT * FROM sub_meter_reading
WHERE fk_sub_meter = $1
ORDER BY reading_date DESC;

-- name: CreateSubMeterReading :one
INSERT INTO sub_meter_reading (
	fk_sub_meter, subid, reading_value, reading_date
) SELECT $1, COALESCE(MAX(subid), 0) + 1, $2, $3
	FROM sub_meter_reading
	WHERE fk_sub_meter = $1
RETURNING *;
