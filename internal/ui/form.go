package ui

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

type SignUpForm struct {
	GeneralErr        string
	Username          string
	UsernameErr       string
	Email             string
	EmailErr          string
	Password          string
	PasswordErr       string
	RepeatPasswordErr string
}

type LogInForm struct {
	GeneralErr  string
	Username    string
	UsernameErr string
	PasswordErr string
}

type MmForm struct {
	GeneralErr             string
	MeterIdentification    string
	MeterIdentificationErr string
	Energy                 string
	EnergyErr              string
	Address                string
	AddressErr             string
	CurrencyCode           string
	CurrencyCodeErr        string
}

type MmSmForm struct {
	GeneralErr             string
	MeterIdentification    string
	MeterIdentificationErr string
	FinBalance             string
	FinBalanceErr          string
}

type SmRdgForm struct {
	GeneralErr string
	RdgVal     string
	RdgValErr  string
	RdgDate    string
	RdgDateErr string
}

type MmBillForm struct {
	GeneralErr    string
	MaxDayDiff    string
	MaxDayDiffErr string
}

func NewMmBillForm() MmBillForm {
	return MmBillForm{MaxDayDiff: "14"}
}

type MmBillPeriodForm struct {
	BeginDate            string
	BeginDateErr         string
	EndDate              string
	EndDateErr           string
	BeginRdgVal          string
	BeginRdgValErr       string
	EndRdgVal            string
	EndRdgValErr         string
	ConsumEnergyPrice    string
	ConsumEnergyPriceErr string
	ServicePrice         string
	ServicePriceErr      string
}

type SmBillForm struct {
	ID                uuid.UUID
	CreatedTs         time.Time
	MeterID           string
	Email             string
	EnergyConsum      float64
	ConsumEnergyPrice float64
	ServicePrice      pgtype.Float8
	AdvancePrice      float64
	FromFinBalance    float64
	ToPay             float64
}

type SmBillForms []*SmBillForm

func (f SmBillForms) Less(i, j int) bool {
	return f[i].CreatedTs.Before(f[j].CreatedTs)
}
func (f SmBillForms) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f SmBillForms) Len() int      { return len(f) }
