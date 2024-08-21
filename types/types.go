package types

import "encoding/json"

type ClientConfig struct {
	Email string
	Usage string
}

type GetConfig struct {
	CIK     string
	Ticker  string
	Doc     string
	Period  int
	RawFile string
}

type ParseConfig struct {
	File   string
	Format string
	Output string
}

// The CompanyFilings struct is used to unmarshal the JSON response from
// the data.sec.gov/submissions/ API endpoint. The JSON response contains
// TODO: flesh out
type CompanyFilings struct {
	Cik     string `json:"cik"`
	Name    string `json:"name"`
	Filings struct {
		Recent struct {
			AccessionNumber []string `json:"accessionNumber"`
			FilingDate      []string `json:"filingDate"`
			Form            []string `json:"form"`
			PrimaryDocument []string `json:"primaryDocument"`
		} `json:"recent"`
	} `json:"filings"`
}

// The Ticker struct is used to unmarshal the JSON response from the
// sec.gov/files/company_tickers.json endpoint.
type Ticker struct {
	Cik  int    `json:"cik_str"`
	Tick string `json:"ticker"`
}

// The CompanyFacts, FactData, UnitData, UnitEntry structs are used to unmarshal the JSON response from the
// https://data.sec.gov/api/xbrl/companyfacts/ endpoint
// TODO: rename the data members like USGAAP, USD
type CompanyFacts struct {
	Cik        int    `json:"cik"`
	EntityName string `json:"entityName"`
	Facts      struct {
		USGAAP map[string]FactData `json:"us-gaap"`
	} `json:"facts"`
}

type FactData struct {
	Label string   `json:"label"`
	Units UnitData `json:"units"`
}

type UnitData struct {
	USD []UnitEntry `json:"USD"`
}

type UnitEntry struct {
	PeriodEnd  string      `json:"end"`
	Value      json.Number `json:"val"`
	FiscalYear int         `json:"fy"`
	ForPeriod  string      `json:"fp"`
	Form       string      `json:"form"`
}

// AI slop
type FinancialData struct {
	Category string
	Year1    string
	Year2    string
	Year3    string
}
