// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: main_meter_billing.sql

package spinusdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const getPreviousSubMeterBillingAdvancePrices = `-- name: GetPreviousSubMeterBillingAdvancePrices :many
SELECT	sub_meter.subid,
	sub_meter_billing.advance_price
FROM (
	SELECT		id
	FROM		main_meter_billing
	WHERE		fk_main_meter = $1
	ORDER BY 	subid DESC
	LIMIT 1
) previous_main_meter_billing
JOIN	sub_meter_billing
	ON previous_main_meter_billing.id = sub_meter_billing.fk_main_billing
JOIN	sub_meter
	ON sub_meter_billing.fk_sub_meter = sub_meter.id
`

type GetPreviousSubMeterBillingAdvancePricesRow struct {
	Subid        int32
	AdvancePrice float64
}

func (q *Queries) GetPreviousSubMeterBillingAdvancePrices(ctx context.Context, mainMeterID pgtype.Int4) ([]GetPreviousSubMeterBillingAdvancePricesRow, error) {
	rows, err := q.db.Query(ctx, getPreviousSubMeterBillingAdvancePrices, mainMeterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetPreviousSubMeterBillingAdvancePricesRow
	for rows.Next() {
		var i GetPreviousSubMeterBillingAdvancePricesRow
		if err := rows.Scan(&i.Subid, &i.AdvancePrice); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMainMeterBillingSubMeters = `-- name: ListMainMeterBillingSubMeters :many
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
ORDER BY subid
`

type ListMainMeterBillingSubMetersRow struct {
	ID                  int32
	Subid               int32
	MeterID             pgtype.Text
	Email               string
	EnergyConsumption   float64
	ConsumedEnergyPrice float64
	ServicePrice        pgtype.Float8
	AdvancePrice        float64
	TotalPrice          float64
}

func (q *Queries) ListMainMeterBillingSubMeters(ctx context.Context, fkMainBilling int32) ([]ListMainMeterBillingSubMetersRow, error) {
	rows, err := q.db.Query(ctx, listMainMeterBillingSubMeters, fkMainBilling)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListMainMeterBillingSubMetersRow
	for rows.Next() {
		var i ListMainMeterBillingSubMetersRow
		if err := rows.Scan(
			&i.ID,
			&i.Subid,
			&i.MeterID,
			&i.Email,
			&i.EnergyConsumption,
			&i.ConsumedEnergyPrice,
			&i.ServicePrice,
			&i.AdvancePrice,
			&i.TotalPrice,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
