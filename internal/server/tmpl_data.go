package server

import spinusdb "github.com/svoboond/spinus/internal/db/sqlc"

type MainMeterTmplData struct {
	ID int32
}

type MainMeterOverviewTmplData struct {
	spinusdb.GetMainMeterRow
	Upper MainMeterTmplData
}

type SubMeterListTmplData struct {
	SubMeters []spinusdb.ListSubMetersRow
	Upper     MainMeterTmplData
}

type SubMeterCreateTmplData struct {
	SubMeterFormData
	Upper MainMeterTmplData
}

type SubMeterTmplData struct {
	MainMeterID int32
	Subid       int32
}

type SubMeterOverviewTmplData struct {
	spinusdb.GetSubMeterRow
	Upper SubMeterTmplData
}

type SubMeterReadingListTmplData struct {
	SubMeterReadings []spinusdb.SubMeterReading
	Upper            SubMeterTmplData
}

type SubMeterReadingCreateTmplData struct {
	SubMeterReadingFormData
	Upper SubMeterTmplData
}

type MainMeterBillingListTmplData struct {
	Upper MainMeterTmplData
}

type MainMeterBillingCreateTmplData struct {
	MainMeterBillingFormData
	Upper MainMeterTmplData
}
