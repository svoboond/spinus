package server

import "github.com/jackc/pgx/v5/pgtype"

type SignUpFormData struct {
	GeneralError        string
	Username            string
	UsernameError       string
	Email               string
	EmailError          string
	Password            string
	PasswordError       string
	RepeatPasswordError string
}

type LogInFormData struct {
	GeneralError  string
	Username      string
	UsernameError string
	Password      string
	PasswordError string
}

type MainMeterFormData struct {
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

type SubMeterFormData struct {
	GeneralError          string
	MeterID               string
	MeterIDError          string
	FinancialBalance      string
	FinancialBalanceError string
}

type SubMeterReadingFormData struct {
	GeneralError      string
	ReadingValue      string
	ReadingValueError string
	ReadingDate       string
	ReadingDateError  string
}

type MainMeterBillingPeriodFormData struct {
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

func NewMainMeterBillingFormData() MainMeterBillingFormData {
	return MainMeterBillingFormData{
		MaxDayDiff:              "14",
		MainMeterBillingPeriods: []*MainMeterBillingPeriodFormData{{}},
	}
}

type MainMeterBillingSubMeterFormData struct {
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

type MainMeterBillingFormData struct {
	GeneralError            string
	MaxDayDiff              string
	MaxDayDiffError         string
	MainMeterBillingPeriods []*MainMeterBillingPeriodFormData
	SubMeterBillings        []*MainMeterBillingSubMeterFormData
	Calculated              bool
}
