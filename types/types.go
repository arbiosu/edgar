package types

type ServerConfig struct {
	Port  int
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

// The Company struct is used to unmarshal the JSON response from
// the data.sec.gov/submissions/ API endpoint.
type Company struct {
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
