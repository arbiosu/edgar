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
	Format  string // JSON or HTML
}

type ParseConfig struct {
	File   string
	Format string
	Output string
}

// The CompanyFilings struct is used to unmarshal the JSON response from
// the data.sec.gov/submissions/ API endpoint. The JSON response contains
// the information needed to download documents from the SEC site.
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
		// get all fact names as a key with their data as the value
		Data map[string]FactData `json:"us-gaap"`
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
	Value      json.Number `json:"val"` // use json.Number because some values are floats
	FiscalYear int         `json:"fy"`
	ForPeriod  string      `json:"fp"`
	Form       string      `json:"form"`
}

// From: https://github.com/Nneoma-Ihueze/SEC-Mapping/blob/main/xbrl_to_fin-statement_mapping.json
// Maps XBRL tags to financial statement
// Used to unmarshal json from above link. Use this info to check for these tags, since every report
// can have slight differences in what's reported, so cannot hardcode tags
type XBRLTags struct {
	Tags struct {
		BalanceSheetItems struct {
			Assets struct {
				CurrentAssets    []string `json:"Current Assets"`
				NonCurrentAssets []string `json:"Non-Current Assets"`
				TotalAssets      []string `json:"Total Assets"`
			} `json:"Assets"`
			Liabilities struct {
				CurrentLiabilities    []string `json:"Current Liabilities"`
				NonCurrentLiabilities []string `json:"Non-Current Liabilities"`
				TotalLiabilities      []string `json:"Total Liabilities"`
			} `json:"Liabilities"`
			Equity                    []string `json:"Equity"`
			TotalLiabilitiesAndEquity []string `json:"Total Liabilities and Equity"`
		} `json:"Balance Sheet Items"`
		IncomeStatementItems struct {
			Revenue             []string `json:"Revenue"`
			CostOfRevenue       []string `json:"Cost of Revenue"`
			GrossProfit         []string `json:"Gross Profit"`
			OperatingExpenses   []string `json:"Operating Expenses"`
			OperatingIncomeLoss []string `json:"Operating Income/Loss"`
			OtherIncomeExpense  []string `json:"Other Income/Expense"`
			IncomeBeforeTax     []string `json:"Income Before Tax"`
			IncomeTax           []string `json:"Income Tax"`
			NetIncomeLoss       []string `json:"Net Income/Loss"`
		} `json:"Income Statement Items"`
		CashFlowStatementItems struct {
			OperatingActivities    []string `json:"Operating Activities"`
			InvestingActivities    []string `json:"Investing Activities"`
			FinancingActivities    []string `json:"Financing Activities"`
			CashAndCashEquivalents []string `json:"Cash and Cash Equivalents"`
		} `json:"Cash Flow Statement Items"`
		OtherComprehensiveIncomeItems []string `json:"Other Comprehensive Income Items"`
		FinancialMetricsAndRatios     []string `json:"Financial Metrics and Ratios"`
		ShareBasedCompensation        []string `json:"Share-Based Compensation"`
		Taxes                         []string `json:"Taxes"`
		Leases                        []string `json:"Leases"`
		DebtAndBorrowings             []string `json:"Debt and Borrowings"`
		IntangibleAssetsAndGoodwill   []string `json:"Intangible Assets and Goodwill"`
		CommitmentsAndContingencies   []string `json:"Commitments and Contingencies"`
		DerivativesAndHedging         []string `json:"Derivatives and Hedging"`
		StockAndEquityRelatedItems    []string `json:"Stock and Equity-related Items"`
		OtherFinancialItems           []string `json:"Other Financial Items"`
	} `json:"Comprehensive Categorization of All Financial Items"`
}

// The Report struct is used to assemble a financial report for a user
type FinancialStatement struct {
	IncomeStatement struct {
		Revenue             []LineItem
		CostOfRevenue       []LineItem
		GrossProfit         []LineItem
		OperatingExpenses   []LineItem
		OperatingIncomeLoss []LineItem
		OtherIncomeExpense  []LineItem
		IncomeBeforeTax     []LineItem
		IncomeTax           []LineItem
		NetIncomeLoss       []LineItem
	}
	BalanceSheet struct {
		Assets struct {
			CurrentAssets    []LineItem
			NonCurrentAssets []LineItem
			TotalAssets      []LineItem
		}
		Liabilities struct {
			CurrentLiabilities    []LineItem
			NonCurrentLiabilities []LineItem
			TotalLiabilities      []LineItem
		}
		Equity                    []LineItem
		TotalLiabilitiesAndEquity []LineItem
	}
	CashFlowStatement struct {
		OperatingActivities    []LineItem
		InvestingActivities    []LineItem
		FinancingActivities    []LineItem
		CashAndCashEquivalents []LineItem
	}
	OtherComprehensiveIncome    []LineItem
	FinancialMetricsAndRatios   []LineItem
	ShareBasedCompensation      []LineItem
	Taxes                       []LineItem
	Leases                      []LineItem
	DebtAndBorrowings           []LineItem
	IntangibleAssetsAndGoodwill []LineItem
	CommitmentsAndContingencies []LineItem
	DerivativesAndHedging       []LineItem
	StockAndEquityRelatedItems  []LineItem
	OtherFinancialItems         []LineItem
}

type LineItem struct {
	Tag  string
	Data []UnitEntry
}
