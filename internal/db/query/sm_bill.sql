-- name: ListSmBills :many
SELECT *
FROM sm_bill
WHERE fk_sm = $1
ORDER BY created_ts DESC;
