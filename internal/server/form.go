package server

import "github.com/jackc/pgx/v5/pgtype"

type SignUpForm struct {
	GeneralError        string
	Username            string
	UsernameError       string
	Email               string
	EmailError          string
	Password            string
	PasswordError       string
	RepeatPasswordError string
}

type LogInForm struct {
	GeneralError  string
	Username      string
	UsernameError string
	Password      string
	PasswordError string
}

type MainMeterForm struct {
	GeneralError      string
	MeterID           string
	MeterIDError      string
	Energy            string
	EnergyError       string
	Address           string
	AddressError      string
	CurrencyCode      string
	CurrencyCodeError string
}

type SubMeterForm struct {
	GeneralError          string
	MeterID               string
	MeterIDError          string
	FinancialBalance      string
	FinancialBalanceError string
}

type SubMeterReadingForm struct {
	GeneralError      string
	ReadingValue      string
	ReadingValueError string
	ReadingDate       string
	ReadingDateError  string
}

func NewMainMeterBillingForm() MainMeterBillingForm {
	return MainMeterBillingForm{
		MaxDayDiff:              "14",
		MainMeterBillingPeriods: []*MainMeterBillingPeriodForm{{}},
		SubMeterBillings:        MainMeterBillingSubMeterForms{},
	}
}

type MainMeterBillingForm struct {
	GeneralError            string
	MaxDayDiff              string
	MaxDayDiffError         string
	MainMeterBillingPeriods []*MainMeterBillingPeriodForm
	SubMeterBillings        MainMeterBillingSubMeterForms
	Calculated              bool
}

type MainMeterBillingPeriodForm struct {
	BeginDate                string
	BeginDateError           string
	EndDate                  string
	EndDateError             string
	BeginReadingValue        string
	BeginReadingValueError   string
	EndReadingValue          string
	EndReadingValueError     string
	ConsumedEnergyPrice      string
	ConsumedEnergyPriceError string
	ServicePrice             string
	ServicePriceError        string
}

type MainMeterBillingSubMeterForm struct {
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

type MainMeterBillingSubMeterForms []*MainMeterBillingSubMeterForm

func (f MainMeterBillingSubMeterForms) Less(i, j int) bool { return f[i].Subid < f[j].Subid }
func (f MainMeterBillingSubMeterForms) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f MainMeterBillingSubMeterForms) Len() int           { return len(f) }
