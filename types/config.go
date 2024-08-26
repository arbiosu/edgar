package types

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	companyFilings = "https://data.sec.gov/submissions/"
	companyFacts   = "https://data.sec.gov/api/xbrl/companyfacts/"
	companyTickers = "https://www.sec.gov/files/company_tickers.json"
)

// Holds the user's email and usage statemenr
// Required for headers to access the SEC API
type ClientConfig struct {
	Email string
	Usage string
}

func (c *ClientConfig) handleClient() {
	err := createDir("config")
	if err != nil {
		fmt.Printf("Error: could not create 'config' directory! (%v)\n", err)
		os.Exit(1)
	}
	bytes, err := json.Marshal(c)
	if err != nil {
		fmt.Printf("Error: could not marshal JSON! (%v)\n", err)
		os.Exit(1)
	}
	err = os.WriteFile("config/config.json", bytes, 0660)
	if err != nil {
		fmt.Printf("Error: could not create config.json! (%v)\n", err)
		os.Exit(1)
	}
}

// Check for a previous client configuration. If the config.json file does not
// exist. TODO: create one
func checkConfig() *ClientConfig {
	fmt.Println("Checking for previous client configuration...")
	c, err := os.ReadFile("./config/config.json")
	if err != nil {
		fmt.Printf("Error: could not read config file: (%v)\n", err)
		os.Exit(1)
	}
	var clientConfig ClientConfig
	err = json.Unmarshal(c, &clientConfig)

	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Config found!")
	return &clientConfig
}

// Downloads company_tickers.json to config/ directory. Returns an error if
// unsuccessful
func (c *ClientConfig) getCompanyTickers() error {
	b, err := c.makeSecRequest("")
	if err != nil {
		return err
	}
	err = os.WriteFile("config/company_tickers.json", b, 0666)
	if err != nil {
		return err
	}
	return nil
}

// Check for company_tickers.json file. If it exists, returns map of tickers
// If it does not exist, make a request and download it from SEC
func (c *ClientConfig) checkCompanyTickers() map[string]int {
	data, err := os.ReadFile("config/company_tickers.json")
	if err != nil {
		fmt.Printf("Error: could not find company_tickers.json! (%v)\n", err)
		// TODO: handle
		fmt.Println("Requesting file...")
		err = c.getCompanyTickers()
		if err != nil {
			fmt.Printf("Error: could not get company_tickers.json! (%v)\n", err)
			fmt.Println("Exiting program...")
			os.Exit(1)
		} else {
			fmt.Println("Success! File company_tickers.json downloaded!")
			// Recursive call to unmarshal JSON
			// TODO: Bug here. File downloads then runs into unexpected end of
			// JSON input. May change to return here and instruct to try again
			c.checkCompanyTickers()
		}
	}
	// TODO: rethink this section, figure out how we want to save the file
	// potentially make it so the json is edited into "ticker":"cik" already
	// instead of making a map from the raw json everytime
	var tickers map[int]Ticker
	err = json.Unmarshal(data, &tickers)
	if err != nil {
		fmt.Printf("Error: could not unmarshal company_tickers.json! (%v)\n", err)
		// TODO: handle
	}
	// edited := editCompanyTickers(tickers)
	edit := func(tickers map[int]Ticker) map[string]int {
		m := make(map[string]int)
		for _, v := range tickers {
			m[v.Tick] = v.Cik
		}
		return m
	}
	edited := edit(tickers)
	return edited
}

// Returns the CompanyFacts struct for a given ticker/CIK
func (c *ClientConfig) getCompanyFacts(url string) *CompanyFacts {
	body, err := c.makeSecRequest(url)
	if err != nil {
		fmt.Printf("Error: could not make SEC Request! (%v)\n", err)
		os.Exit(1)
	}
	var cf CompanyFacts
	err = json.Unmarshal(body, &cf)
	if err != nil {
		fmt.Printf("Error: could not unmarshal JSON! (%V)\n", err)
		os.Exit(1)
	}
	return &cf
}

// Makes a GET request to the given URL and returns the response body.
func (c *ClientConfig) makeSecRequest(url string) ([]byte, error) {
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = http.Header{
		"User-Agent":   {c.Usage + " " + c.Email},
		"Content-Type": {"application/json"},
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type GetConfig struct {
	CIK     string
	Ticker  string
	Doc     string
	Period  int
	RawFile string
	Format  string // JSON or HTML
}

func handleGet(g *GetConfig) {
	c := checkConfig()
	if g.CIK == "" {
		tickers := c.checkCompanyTickers()
		cik, ok := tickers[g.Ticker]
		if !ok {
			fmt.Printf("Error: ticker not found! Exiting program.\n")
			os.Exit(1)
		}
		cikStr := strconv.Itoa(cik)
		padded := zeroPad(cikStr)
		g.CIK = padded
	}
	var url string
	switch g.Format {
	case "json":
		url = assembleUrl(g.CIK, companyFacts)
		facts := c.getCompanyFacts(url)
		xbrl := getXBRLTags()
		r, err := g.assembleReport(facts, xbrl)
		if err != nil {
			fmt.Printf("Error: could not assemble company report! (%v)\n", err)
			os.Exit(1)
		}
		if g.RawFile == "" {
			g.RawFile = g.Ticker + "_company_facts"
		}
		err = g.downloadJSON(r)
		if err != nil {
			fmt.Printf("Error: could not download company report! (%v)\n", err)
		}
	case "html":
		url = assembleUrl(g.CIK, companyFilings)
		urls := g.getFileUrls(url, c)
		err := g.downloadFiles(urls, c)
		if err != nil {
			fmt.Printf("Error: Failed to download files! (%v)\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("URL: %s\nCIK: %s\nTicker: %s\nFormat: %s\nFiling(s): %s\nPeriod: %d\n", url, g.CIK, g.Ticker, g.Format, g.Doc, g.Period)
	fmt.Println("Your desired report(s) are located in the app/ directory. Thanks for using edgar!")
	os.Exit(0)
}

// Gets the URLs of the desired filings
func (g *GetConfig) getFileUrls(url string, c *ClientConfig) []string {
	body, err := c.makeSecRequest(url)
	if err != nil {
		fmt.Printf("Error: could not make SEC Request! (%v)\n", err)
		// TODO: handle
	}
	var cf CompanyFilings
	err = json.Unmarshal(body, &c)
	if err != nil {
		fmt.Printf("Error: could not unmarshal JSON! (%V)\n", err)
		// TODO: handle
	}
	// Iterate over the Form slice to find the index of the desired filings.
	// Get the accession number and the primary document at the asscoiated index.
	// Assemble the URLs to retrieve the desired filings.
	newUrl := "https://www.sec.gov/Archives/edgar/data/"
	urls := make([]string, 0)
	for i, v := range cf.Filings.Recent.Form {
		// TODO: Validate g.Period with c.Filings.Recent.FilingDate[i]
		// Ensure g.Doc is correct ie: 10-K, 10-Q
		if v == g.Doc {
			// strip '-' from accession number
			re := regexp.MustCompile(`-`)
			cleaned := re.ReplaceAllString(cf.Filings.Recent.AccessionNumber[i], "")
			urls = append(urls, newUrl+cf.Cik+"/"+cleaned+"/"+cf.Filings.Recent.PrimaryDocument[i])
		}
	}
	return urls
}

// TODO: rethink downloadFiles and downloadJSON
func (g *GetConfig) downloadFiles(urls []string, c *ClientConfig) error {
	err := createDir("app/" + g.Ticker + "/")
	if err != nil {
		fmt.Printf("Error: could not create 'app' directory! (%v)\n", err)
		os.Exit(1)
		// TODO: handle
	}
	for i := 0; i < len(urls); i++ {
		body, err := c.makeSecRequest(urls[i])
		if err != nil {
			fmt.Printf("Error: could not request %s! (%v)\n", urls[i], err)
			return err
		}
		err = os.WriteFile("./app/"+g.Ticker+"/"+g.Doc+".html", body, 0666)
		if err != nil {
			fmt.Printf("Error: could not write file! (%v)\n", err)
			return err
		}
		g.RawFile = "./app/" + g.Ticker + "/" + g.Doc + ".html"
	}
	return nil
}

func (g *GetConfig) downloadJSON(r *FinancialStatement) error {
	err := createDir("app/" + g.Ticker + "/")
	if err != nil {
		fmt.Printf("Error: could not create 'app' directory! (%v)\n", err)
		return err
	}
	b, err := json.MarshalIndent(r, "", "	")
	if err != nil {
		fmt.Printf("Error: could not marshal financial statement! (%v)\n", err)
		return err
	}
	err = os.WriteFile("./app/"+g.Ticker+"/"+g.RawFile+".json", b, 0770)
	if err != nil {
		fmt.Printf("Error: could not write file to app dir! (%v)\n", err)
		return err
	}
	return nil
}

// Assemble the financial statement report
// TODO: FIX!!
func (g *GetConfig) assembleReport(f *CompanyFacts, xbrl *XBRLTags) (*FinancialStatement, error) {
	report := &FinancialStatement{}
	g.assembleBalanceSheet(f, xbrl, report)
	g.assembleIncomeStatement(f, xbrl, report)
	g.assembleCashFlowStatement(f, xbrl, report)
	return report, nil
}

// Assembles the balance sheet
// TODO: better way to do this?
func (g *GetConfig) assembleBalanceSheet(f *CompanyFacts, xbrl *XBRLTags, r *FinancialStatement) {
	data := f.Facts.Data
	// Assemble Assets
	currentAssets := xbrl.Tags.BalanceSheetItems.Assets.CurrentAssets
	nonCurrentAssets := xbrl.Tags.BalanceSheetItems.Assets.NonCurrentAssets
	totalAssets := xbrl.Tags.BalanceSheetItems.Assets.TotalAssets
	iterateTags(data, currentAssets, &r.BalanceSheet.Assets.CurrentAssets, g)
	iterateTags(data, nonCurrentAssets, &r.BalanceSheet.Assets.NonCurrentAssets, g)
	iterateTags(data, totalAssets, &r.BalanceSheet.Assets.TotalAssets, g)
	// Assemble liabilities
	currLiabilities := xbrl.Tags.BalanceSheetItems.Liabilities.CurrentLiabilities
	nonCurrLiabilities := xbrl.Tags.BalanceSheetItems.Liabilities.NonCurrentLiabilities
	totalLiabilities := xbrl.Tags.BalanceSheetItems.Liabilities.TotalLiabilities
	iterateTags(data, currLiabilities, &r.BalanceSheet.Liabilities.CurrentLiabilities, g)
	iterateTags(data, nonCurrLiabilities, &r.BalanceSheet.Liabilities.NonCurrentLiabilities, g)
	iterateTags(data, totalLiabilities, &r.BalanceSheet.Liabilities.TotalLiabilities, g)
	// Assemble equity
	equity := xbrl.Tags.BalanceSheetItems.Equity
	totalLiabilitiesEquity := xbrl.Tags.BalanceSheetItems.TotalLiabilitiesAndEquity
	iterateTags(data, equity, &r.BalanceSheet.Equity, g)
	iterateTags(data, totalLiabilitiesEquity, &r.BalanceSheet.TotalLiabilitiesAndEquity, g)
}

func (g *GetConfig) assembleIncomeStatement(f *CompanyFacts, xbrl *XBRLTags, r *FinancialStatement) {
	data := f.Facts.Data
	revenue := xbrl.Tags.IncomeStatementItems.Revenue
	cogs := xbrl.Tags.IncomeStatementItems.CostOfRevenue
	gp := xbrl.Tags.IncomeStatementItems.GrossProfit
	opex := xbrl.Tags.IncomeStatementItems.OperatingExpenses
	opIncomeLos := xbrl.Tags.IncomeStatementItems.OperatingIncomeLoss
	other := xbrl.Tags.IncomeStatementItems.OtherIncomeExpense
	incomeBeforeTax := xbrl.Tags.IncomeStatementItems.IncomeBeforeTax
	tax := xbrl.Tags.IncomeStatementItems.IncomeTax
	ni := xbrl.Tags.IncomeStatementItems.NetIncomeLoss
	iterateTags(data, revenue, &r.IncomeStatement.Revenue, g)
	iterateTags(data, cogs, &r.IncomeStatement.CostOfRevenue, g)
	iterateTags(data, gp, &r.IncomeStatement.GrossProfit, g)
	iterateTags(data, opex, &r.IncomeStatement.OperatingExpenses, g)
	iterateTags(data, opIncomeLos, &r.IncomeStatement.OperatingIncomeLoss, g)
	iterateTags(data, other, &r.IncomeStatement.OtherIncomeExpense, g)
	iterateTags(data, incomeBeforeTax, &r.IncomeStatement.IncomeBeforeTax, g)
	iterateTags(data, tax, &r.IncomeStatement.IncomeBeforeTax, g)
	iterateTags(data, ni, &r.IncomeStatement.NetIncomeLoss, g)
}

func (g *GetConfig) assembleCashFlowStatement(f *CompanyFacts, xbrl *XBRLTags, r *FinancialStatement) {
	data := f.Facts.Data
	opActivities := xbrl.Tags.CashFlowStatementItems.OperatingActivities
	investingActivities := xbrl.Tags.CashFlowStatementItems.InvestingActivities
	financingActivities := xbrl.Tags.CashFlowStatementItems.FinancingActivities
	// TODO: ?
	cash := xbrl.Tags.CashFlowStatementItems.CashAndCashEquivalents
	iterateTags(data, opActivities, &r.CashFlowStatement.OperatingActivities, g)
	iterateTags(data, investingActivities, &r.CashFlowStatement.InvestingActivities, g)
	iterateTags(data, financingActivities, &r.CashFlowStatement.FinancingActivities, g)
	iterateTags(data, cash, &r.CashFlowStatement.CashAndCashEquivalents, g)
}

func iterateTags(d map[string]FactData, item []string, l *[]LineItem, g *GetConfig) {
	for i := 0; i < len(item); i++ {
		factData, ok := d[item[i]]
		if ok {
			relevant := g.findRelevantUnitEntries(&factData.Units.USD)
			newLi := &LineItem{Tag: factData.Label, Data: *relevant}
			*l = append(*l, *newLi)
		}
	}
}

func (g *GetConfig) findRelevantUnitEntries(entries *[]UnitEntry) *[]UnitEntry {
	var relevantEntries []UnitEntry
	for _, v := range *entries {
		if v.Form == g.Doc && v.FiscalYear == g.Period {
			relevantEntries = append(relevantEntries, v)
		}
	}
	return &relevantEntries
}

// Gets the XBRL tags associated with income statements, balance sheets and cash flow statements
func getXBRLTags() *XBRLTags {
	b, err := os.ReadFile("xbrl_to_fin-statement_mapping.json")
	if err != nil {
		fmt.Printf("Error: could not read xbrl mapping file! (%v)", err)
		os.Exit(1)
	}
	var xbrl XBRLTags
	err = json.Unmarshal(b, &xbrl)
	if err != nil {
		fmt.Printf("Error: could not unmarshal xbrl mapping file! (%v)", err)
		os.Exit(1)
	}
	return &xbrl
}

// Creates a directory
func createDir(name string) error {
	var err error
	// if the given directory name contains subdirectories, use MkdirAll
	if filepath.Dir(name) != "." {
		err = os.MkdirAll(name, os.ModePerm)
	} else {
		err = os.Mkdir(name, os.ModePerm)
	}
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

// Adds leading zeroes to a CIK string to make it 10 characters long and
// compatible with the SEC API.
func zeroPad(cik string) string {
	for len(cik) < 10 {
		cik = "0" + cik
	}
	cik = "CIK" + cik
	return cik
}

// Assemble the url to get company info.
func assembleUrl(cik string, url string) string {
	return url + cik + ".json"
}
