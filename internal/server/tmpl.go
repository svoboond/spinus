package server

import spinusdb "github.com/svoboond/spinus/internal/db/sqlc"

type MmUpperTmpl struct {
	ID int32
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
	MmID  int32
	Subid int32
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

type MmBillListTmpl struct {
	MmBills []spinusdb.MmBill
	Upper   MmUpperTmpl
}

type MmBillCreateTmpl struct {
	MmBillForm
	Upper MmUpperTmpl
}

type MmBillUpperTmpl struct {
	Subid int32
}

type MmBillOverviewTmpl struct {
	spinusdb.MmBill
	Upper     MmUpperTmpl
	BillUpper MmBillUpperTmpl
}
