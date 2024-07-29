package server

import (
	"github.com/google/uuid"
	spinusdb "github.com/svoboond/spinus/internal/db/sqlc"
)

type MmUpperTmpl struct {
	MmID uuid.UUID
}

type MmOverviewTmpl struct {
	spinusdb.GetMmRow
	Upper MmUpperTmpl
}

type SmListTmpl struct {
	Sms   []spinusdb.ListSmsRow
	Upper MmUpperTmpl
}

type SmCreateTmpl struct {
	SmForm
	Upper MmUpperTmpl
}

type SmUpperTmpl struct {
	MmUpperTmpl
	SmID uuid.UUID
}

type SmOverviewTmpl struct {
	spinusdb.GetSmRow
	Upper SmUpperTmpl
}

type SmRdgListTmpl struct {
	SmRdgs []spinusdb.SmRdg
	Upper  SmUpperTmpl
}

type SmRdgCreateTmpl struct {
	SmRdgForm
	Upper SmUpperTmpl
}

type SmBillListTmpl struct {
	SmBills []spinusdb.SmBill
	Upper   SmUpperTmpl
}

type MmBillListTmpl struct {
	MmBills []spinusdb.MmBill
	Upper   MmUpperTmpl
}

type MmBillCreateTmpl struct {
	MmBillForm
	Upper MmUpperTmpl
}

type MmBillUpperTmpl struct {
	MmUpperTmpl
	MmBillID uuid.UUID
}

type MmBillOverviewTmpl struct {
	spinusdb.GetMmBillRow
	Upper MmBillUpperTmpl
}

type MmBillSmListTmpl struct {
	MmBillSms []spinusdb.ListMmBillSmsRow
	Upper     MmBillUpperTmpl
}

type MmBillPeriodListTmpl struct {
	MmBillPeriods []spinusdb.MmBillPeriod
	Upper         MmBillUpperTmpl
}
