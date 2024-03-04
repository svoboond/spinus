package server

import spinusdb "github.com/svoboond/spinus/internal/db/sqlc"

type MainMeterTmplData struct {
	ID int32
}

type MainMeterGeneralTmplData struct {
	spinusdb.MainMeter
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
