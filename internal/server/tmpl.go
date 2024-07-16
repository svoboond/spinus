package server

import spinusdb "github.com/svoboond/spinus/internal/db/sqlc"

type MmTmpl struct {
	ID int32
}

type MmOverviewTmpl struct {
	spinusdb.GetMmRow
	Upper MmTmpl
}

type SmListTmpl struct {
	Sms   []spinusdb.ListSmsRow
	Upper MmTmpl
}

type SmCreateTmpl struct {
	SmForm
	Upper MmTmpl
}

type SmTmpl struct {
	MmID  int32
	Subid int32
}

type SmOverviewTmpl struct {
	spinusdb.GetSmRow
	Upper SmTmpl
}

type SmRdgListTmpl struct {
	SmRdgs []spinusdb.SmRdg
	Upper  SmTmpl
}

type SmRdgCreateTmpl struct {
	SmRdgForm
	Upper SmTmpl
}

type MmBillListTmpl struct {
	MmBills []spinusdb.MmBill
	Upper   MmTmpl
}

type MmBillCreateTmpl struct {
	MmBillForm
	Upper MmTmpl
}
