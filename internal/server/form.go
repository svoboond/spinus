package server

import "github.com/jackc/pgx/v5/pgtype"

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
	Password    string
	PasswordErr string
}

type MmForm struct {
	GeneralErr      string
	MeterID         string
	MeterIDErr      string
	Energy          string
	EnergyErr       string
	Address         string
	AddressErr      string
	CurrencyCode    string
	CurrencyCodeErr string
}

type SmForm struct {
	GeneralErr    string
	MeterID       string
	MeterIDErr    string
	FinBalance    string
	FinBalanceErr string
}

type SmRdgForm struct {
	GeneralErr string
	RdgVal     string
	RdgValErr  string
	RdgDate    string
	RdgDateErr string
}

func NewMmBillForm() MmBillForm {
	return MmBillForm{
		MaxDayDiff:    "14",
		MmBillPeriods: []*MmBillPeriodForm{{}},
		SmBills:       SmBillForms{},
	}
}

type MmBillForm struct {
	GeneralErr    string
	MaxDayDiff    string
	MaxDayDiffErr string
	MmBillPeriods []*MmBillPeriodForm
	SmBills       SmBillForms
	Calculated    bool
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
	ID                int32
	Subid             int32
	MeterID           pgtype.Text
	Email             string
	EnergyConsum      float64
	ConsumEnergyPrice float64
	ServicePrice      pgtype.Float8
	AdvancePrice      float64
	FromFinBalance    float64
	ToPay             float64
}

type SmBillForms []*SmBillForm

func (f SmBillForms) Less(i, j int) bool { return f[i].Subid < f[j].Subid }
func (f SmBillForms) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f SmBillForms) Len() int           { return len(f) }
