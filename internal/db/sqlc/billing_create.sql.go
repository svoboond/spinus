// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: billing_create.sql

package spinusdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createMainMeterBilling = `-- name: CreateMainMeterBilling :one
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
RETURNING id, fk_main_meter, subid, max_day_diff, begin_date, end_date, energy_consumption, consumed_energy_price, service_price, advance_price, from_financial_balance, to_pay, status
`

type CreateMainMeterBillingParams struct {
	FkMainMeter          int32
	MaxDayDiff           int32
	BeginDate            pgtype.Date
	EndDate              pgtype.Date
	EnergyConsumption    float64
	ConsumedEnergyPrice  float64
	ServicePrice         pgtype.Float8
	AdvancePrice         float64
	FromFinancialBalance float64
	ToPay                float64
	Status               MainMeterBillingStatus
}

func (q *Queries) CreateMainMeterBilling(ctx context.Context, arg CreateMainMeterBillingParams) (MainMeterBilling, error) {
	row := q.db.QueryRow(ctx, createMainMeterBilling,
		arg.FkMainMeter,
		arg.MaxDayDiff,
		arg.BeginDate,
		arg.EndDate,
		arg.EnergyConsumption,
		arg.ConsumedEnergyPrice,
		arg.ServicePrice,
		arg.AdvancePrice,
		arg.FromFinancialBalance,
		arg.ToPay,
		arg.Status,
	)
	var i MainMeterBilling
	err := row.Scan(
		&i.ID,
		&i.FkMainMeter,
		&i.Subid,
		&i.MaxDayDiff,
		&i.BeginDate,
		&i.EndDate,
		&i.EnergyConsumption,
		&i.ConsumedEnergyPrice,
		&i.ServicePrice,
		&i.AdvancePrice,
		&i.FromFinancialBalance,
		&i.ToPay,
		&i.Status,
	)
	return i, err
}

const createMainMeterBillingPeriod = `-- name: CreateMainMeterBillingPeriod :one
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
RETURNING id, fk_main_billing, subid, begin_date, end_date, begin_reading_value, end_reading_value, energy_consumption, consumed_energy_price, service_price, advance_price, total_price
`

type CreateMainMeterBillingPeriodParams struct {
	FkMainBilling       int32
	BeginDate           pgtype.Date
	EndDate             pgtype.Date
	BeginReadingValue   float64
	EndReadingValue     float64
	EnergyConsumption   float64
	ConsumedEnergyPrice float64
	ServicePrice        pgtype.Float8
	AdvancePrice        float64
	TotalPrice          float64
}

func (q *Queries) CreateMainMeterBillingPeriod(ctx context.Context, arg CreateMainMeterBillingPeriodParams) (MainMeterBillingPeriod, error) {
	row := q.db.QueryRow(ctx, createMainMeterBillingPeriod,
		arg.FkMainBilling,
		arg.BeginDate,
		arg.EndDate,
		arg.BeginReadingValue,
		arg.EndReadingValue,
		arg.EnergyConsumption,
		arg.ConsumedEnergyPrice,
		arg.ServicePrice,
		arg.AdvancePrice,
		arg.TotalPrice,
	)
	var i MainMeterBillingPeriod
	err := row.Scan(
		&i.ID,
		&i.FkMainBilling,
		&i.Subid,
		&i.BeginDate,
		&i.EndDate,
		&i.BeginReadingValue,
		&i.EndReadingValue,
		&i.EnergyConsumption,
		&i.ConsumedEnergyPrice,
		&i.ServicePrice,
		&i.AdvancePrice,
		&i.TotalPrice,
	)
	return i, err
}

const createSubMeterBilling = `-- name: CreateSubMeterBilling :one
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
RETURNING id, fk_sub_meter, fk_main_billing, subid, energy_consumption, consumed_energy_price, service_price, advance_price, from_financial_balance, to_pay, status
`

type CreateSubMeterBillingParams struct {
	FkSubMeter           int32
	FkMainBilling        int32
	EnergyConsumption    float64
	ConsumedEnergyPrice  float64
	ServicePrice         pgtype.Float8
	AdvancePrice         float64
	FromFinancialBalance float64
	ToPay                float64
	Status               SubMeterBillingStatus
}

func (q *Queries) CreateSubMeterBilling(ctx context.Context, arg CreateSubMeterBillingParams) (SubMeterBilling, error) {
	row := q.db.QueryRow(ctx, createSubMeterBilling,
		arg.FkSubMeter,
		arg.FkMainBilling,
		arg.EnergyConsumption,
		arg.ConsumedEnergyPrice,
		arg.ServicePrice,
		arg.AdvancePrice,
		arg.FromFinancialBalance,
		arg.ToPay,
		arg.Status,
	)
	var i SubMeterBilling
	err := row.Scan(
		&i.ID,
		&i.FkSubMeter,
		&i.FkMainBilling,
		&i.Subid,
		&i.EnergyConsumption,
		&i.ConsumedEnergyPrice,
		&i.ServicePrice,
		&i.AdvancePrice,
		&i.FromFinancialBalance,
		&i.ToPay,
		&i.Status,
	)
	return i, err
}

const createSubMeterBillingPeriod = `-- name: CreateSubMeterBillingPeriod :one
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
RETURNING id, fk_sub_billing, fk_main_billing_period, energy_consumption, consumed_energy_price, service_price, advance_price, total_price
`

type CreateSubMeterBillingPeriodParams struct {
	FkSubBilling        int32
	FkMainBillingPeriod int32
	EnergyConsumption   float64
	ConsumedEnergyPrice float64
	ServicePrice        pgtype.Float8
	AdvancePrice        float64
	TotalPrice          float64
}

func (q *Queries) CreateSubMeterBillingPeriod(ctx context.Context, arg CreateSubMeterBillingPeriodParams) (SubMeterBillingPeriod, error) {
	row := q.db.QueryRow(ctx, createSubMeterBillingPeriod,
		arg.FkSubBilling,
		arg.FkMainBillingPeriod,
		arg.EnergyConsumption,
		arg.ConsumedEnergyPrice,
		arg.ServicePrice,
		arg.AdvancePrice,
		arg.TotalPrice,
	)
	var i SubMeterBillingPeriod
	err := row.Scan(
		&i.ID,
		&i.FkSubBilling,
		&i.FkMainBillingPeriod,
		&i.EnergyConsumption,
		&i.ConsumedEnergyPrice,
		&i.ServicePrice,
		&i.AdvancePrice,
		&i.TotalPrice,
	)
	return i, err
}

const getSubMeterReadings = `-- name: GetSubMeterReadings :many
WITH	selected_sub_meter AS (
	SELECT	sub_meter.id
	FROM	sub_meter
	WHERE	fk_main_meter = $1
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
	WHERE		sub_meter_reading.reading_date > $2
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
	$3 AND $2
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
	WHERE		sub_meter_reading.reading_date < $3
	GROUP BY	selected_sub_meter.id
) earlier_reading
ON		selected_sub_meter.id = earlier_reading.sub_meter_id
LEFT JOIN	sub_meter_reading
ON		earlier_reading.sub_meter_id = sub_meter_reading.fk_sub_meter AND
		earlier_reading.reading_date = sub_meter_reading.reading_date
ORDER BY 	reading_date DESC NULLS LAST
`

type GetSubMeterReadingsParams struct {
	FkMainMeter int32
	DateMax     pgtype.Date
	DateMin     pgtype.Date
}

type GetSubMeterReadingsRow struct {
	SubMeterID   int32
	ReadingValue pgtype.Float8
	ReadingDate  pgtype.Date
}

func (q *Queries) GetSubMeterReadings(ctx context.Context, arg GetSubMeterReadingsParams) ([]GetSubMeterReadingsRow, error) {
	rows, err := q.db.Query(ctx, getSubMeterReadings, arg.FkMainMeter, arg.DateMax, arg.DateMin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetSubMeterReadingsRow
	for rows.Next() {
		var i GetSubMeterReadingsRow
		if err := rows.Scan(&i.SubMeterID, &i.ReadingValue, &i.ReadingDate); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
