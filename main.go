package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/arbiosu/edgar/types"
)

const (
	companyFilings = "https://data.sec.gov/submissions/"
	companyFacts   = "https://data.sec.gov/api/xbrl/companyfacts/"
	companyTickers = "https://www.sec.gov/files/company_tickers.json"
)

func setupFlags(c *types.ClientConfig, g *types.GetConfig, p *types.ParseConfig) map[string]*flag.FlagSet {
	sh := "(shorthand)"
	now := time.Now()
	year := now.Year()

	client := flag.NewFlagSet("client", flag.ExitOnError)
	client.StringVar(&c.Email, "email", "hello@example.com", "Your email address")
	client.StringVar(&c.Email, "e", "hello@example.com", "Your email address"+sh)
	client.StringVar(&c.Usage, "usage", "personal use", "Usage statement")
	client.StringVar(&c.Usage, "u", "personal use", "Usage statement"+sh)

	get := flag.NewFlagSet("get", flag.ExitOnError)
	get.StringVar(&g.CIK, "cik", "", "CIK number")
	get.StringVar(&g.Ticker, "ticker", "", "Stock ticker")
	get.StringVar(&g.Ticker, "t", "", "Stock ticker"+sh)
	get.StringVar(&g.Doc, "doc", "10-K", "Desired document (10-K, 10-Q)")
	get.StringVar(&g.Doc, "d", "10-K", "Desired document"+sh)
	get.IntVar(&g.Period, "period", year, "Time period")
	get.IntVar(&g.Period, "p", year, "Time period"+sh)
	get.StringVar(&g.RawFile, "save", "edgar.html", "Name of the raw file to be be saved")
	get.StringVar(&g.RawFile, "s", "edgar.html", "Name of the raw file to be saved"+sh)
	get.StringVar(&g.Format, "format", "html", "Choose whether to download html files or get a JSON report")
	get.StringVar(&g.Format, "f", "html", "Choose whether to download html files or get a JSON report")

	// TODO: fix
	parse := flag.NewFlagSet("parse", flag.ExitOnError)
	parse.StringVar(&p.File, "file", "edgar", "Name of the raw file to be parsed")
	parse.StringVar(&p.Format, "format", "html", "Format of the parsed file")
	parse.StringVar(&p.Format, "f", "html", "Format of the parsed file")
	parse.StringVar(&p.Output, "output", p.File+"."+p.Format, "Name of the output file")
	parse.StringVar(&p.Output, "o", p.File+"."+p.Format, "Name of the output file")

	m := make(map[string]*flag.FlagSet)
	m["client"] = client
	m["get"] = get
	m["parse"] = parse

	return m
}

func main() {
	var clientConfig types.ClientConfig
	var getConfig types.GetConfig
	var parseConfig types.ParseConfig
	m := setupFlags(&clientConfig, &getConfig, &parseConfig)

	if len(os.Args) < 2 {
		fmt.Println("Error: expected 'client' or 'get' subcommands. Exiting...")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "client":
		m["client"].Parse(os.Args[2:])
		handleClient(&clientConfig)
		fmt.Printf("EDGAR Client Configuration: %+v\n", clientConfig)
	case "get":
		m["get"].Parse(os.Args[2:])
		handleGet(&getConfig)
	case "parse":
		m["parse"].Parse(os.Args[2:])
	default:
		fmt.Println("Expected 'client' or 'get' subcommands")
		os.Exit(1)
	}
}

func handleClient(config *types.ClientConfig) {
	err := createDir("config")
	if err != nil {
		fmt.Printf("Error: could not create 'config' directory! (%v)\n", err)
		os.Exit(1)
	}
	bytes, err := json.Marshal(config)
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

func handleGet(g *types.GetConfig) {
	clientConfig := checkConfig()
	if g.CIK == "" {
		tickers := checkCompanyTickers(clientConfig)
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
		facts := getCompanyFacts(url, clientConfig)
		xbrl := getXBRLTags()
		r, err := assembleReport(facts, xbrl)
		if err != nil {
			fmt.Printf("Error: could not assemble company report! (%v)\n", err)
			os.Exit(1)
		}
		err = downloadJSON(g, r)
		if err != nil {
			fmt.Printf("Error: could not download company report! (%v)\n", err)
		}
	case "html":
		url = assembleUrl(g.CIK, companyFilings)
		urls := getFileUrls(url, clientConfig, g)
		err := downloadFiles(urls, clientConfig, g)
		if err != nil {
			fmt.Printf("Error: Failed to download files! (%v)\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Ticker: %s\nFormat: %s\nFiling(s): %s\nPeriod: %d\n", g.Ticker, g.Format, g.Doc, g.Period)
	fmt.Println("Your desired report(s) are located in the app/ directory. Thanks for using edgar!")
	os.Exit(0)
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

// Check for a previous client configuration. If the config.json file does not
// exist. TODO: create one
func checkConfig() *types.ClientConfig {
	fmt.Println("Checking for previous client configuration...")
	c, err := os.ReadFile("./config/config.json")
	if err != nil {
		fmt.Printf("Error: could not read config file: (%v)\n", err)
		os.Exit(1)
	}
	var clientConfig types.ClientConfig
	err = json.Unmarshal(c, &clientConfig)

	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Config found!")
	return &clientConfig
}

// Makes a GET request to the given URL and returns the response body.
func makeSecRequest(url string, c *types.ClientConfig) ([]byte, error) {
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

// Gets the URLs of the desired filings
func getFileUrls(url string, c *types.ClientConfig, g *types.GetConfig) []string {
	body, err := makeSecRequest(url, c)
	if err != nil {
		fmt.Printf("Error: could not make SEC Request! (%v)\n", err)
		// TODO: handle
	}
	var cf types.CompanyFilings
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
func downloadFiles(urls []string, c *types.ClientConfig, g *types.GetConfig) error {
	err := createDir("app/" + g.Ticker + "/")
	if err != nil {
		fmt.Printf("Error: could not create 'app' directory! (%v)\n", err)
		os.Exit(1)
		// TODO: handle
	}
	for i := 0; i < len(urls); i++ {
		body, err := makeSecRequest(urls[i], c)
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

func downloadJSON(g *types.GetConfig, r *types.FinancialStatement) error {
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
	err = os.WriteFile("./app/"+g.Ticker+"/"+g.Doc+".json", b, 0770)
	if err != nil {
		fmt.Printf("Error: could not write file to app dir! (%v)\n", err)
		return err
	}
	return nil
}

// Downloads company_tickers.json to config/ directory. Returns an error if
// unsuccessful
func getCompanyTickers(c *types.ClientConfig) error {
	b, err := makeSecRequest(companyTickers, c)
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
func checkCompanyTickers(c *types.ClientConfig) map[string]int {
	data, err := os.ReadFile("config/company_tickers.json")
	if err != nil {
		fmt.Printf("Error: could not find company_tickers.json! (%v)\n", err)
		// TODO: handle
		fmt.Println("Requesting file...")
		err = getCompanyTickers(c)
		if err != nil {
			fmt.Printf("Error: could not get company_tickers.json! (%v)\n", err)
			fmt.Println("Exiting program...")
			os.Exit(1)
		} else {
			fmt.Println("Success! File company_tickers.json downloaded!")
			// Recursive call to unmarshal JSON
			// TODO: Bug here. File downloads then runs into unexpected end of
			// JSON input. May change to return here and instruct to try again
			checkCompanyTickers(c)
		}
	}
	// TODO: rethink this section, figure out how we want to save the file
	// potentially make it so the json is edited into "ticker":"cik" already
	// instead of making a map from the raw json everytime
	var tickers map[int]types.Ticker
	err = json.Unmarshal(data, &tickers)
	if err != nil {
		fmt.Printf("Error: could not unmarshal company_tickers.json! (%v)\n", err)
		// TODO: handle
	}
	// edited := editCompanyTickers(tickers)
	edit := func(tickers map[int]types.Ticker) map[string]int {
		m := make(map[string]int)
		for _, v := range tickers {
			m[v.Tick] = v.Cik
		}
		return m
	}
	edited := edit(tickers)
	return edited
}

/* Edits the original company_tickers.json to something more readable
// TODO: decide whether to make this separate function or keep it in checkCompanyTickers
func editCompanyTickers(tickers map[int]types.Ticker) map[string]int {
	m := make(map[string]int)
	for _, v := range tickers {
		m[v.Tick] = v.Cik
	}
	return m
}
*/

// Gets the XBRL tags associated with income statements, balance sheets and cash flow statements
func getXBRLTags() *types.XBRLTags {
	b, err := os.ReadFile("xbrl_to_fin-statement_mapping.json")
	if err != nil {
		fmt.Printf("Error: could not read xbrl mapping file! (%v)", err)
		os.Exit(1)
	}
	var xbrl types.XBRLTags
	err = json.Unmarshal(b, &xbrl)
	if err != nil {
		fmt.Printf("Error: could not unmarshal xbrl mapping file! (%v)", err)
		os.Exit(1)
	}
	return &xbrl
}

// Returns the CompanyFacts struct for a given ticker/CIK
func getCompanyFacts(url string, c *types.ClientConfig) *types.CompanyFacts {
	body, err := makeSecRequest(url, c)
	if err != nil {
		fmt.Printf("Error: could not make SEC Request! (%v)\n", err)
		os.Exit(1)
	}
	var cf types.CompanyFacts
	err = json.Unmarshal(body, &cf)
	if err != nil {
		fmt.Printf("Error: could not unmarshal JSON! (%V)\n", err)
		os.Exit(1)
	}
	return &cf
}

// Assemble the financial statement report
// TODO: FIX!!
func assembleReport(f *types.CompanyFacts, xbrl *types.XBRLTags) (*types.FinancialStatement, error) {
	report := &types.FinancialStatement{}
	assembleBalanceSheet(f, xbrl, report)
	assembleIncomeStatement(f, xbrl, report)
	assembleCashFlowStatement(f, xbrl, report)
	return report, nil
}

// Assembles the balance sheet
// TODO: better way to do this?
func assembleBalanceSheet(f *types.CompanyFacts, xbrl *types.XBRLTags, r *types.FinancialStatement) {
	data := f.Facts.Data
	// Assemble Assets
	currentAssets := xbrl.Tags.BalanceSheetItems.Assets.CurrentAssets
	nonCurrentAssets := xbrl.Tags.BalanceSheetItems.Assets.NonCurrentAssets
	totalAssets := xbrl.Tags.BalanceSheetItems.Assets.TotalAssets
	iterateTags(data, currentAssets, &r.BalanceSheet.Assets.CurrentAssets)
	iterateTags(data, nonCurrentAssets, &r.BalanceSheet.Assets.NonCurrentAssets)
	iterateTags(data, totalAssets, &r.BalanceSheet.Assets.TotalAssets)
	// Assemble liabilities
	currLiabilities := xbrl.Tags.BalanceSheetItems.Liabilities.CurrentLiabilities
	nonCurrLiabilities := xbrl.Tags.BalanceSheetItems.Liabilities.NonCurrentLiabilities
	totalLiabilities := xbrl.Tags.BalanceSheetItems.Liabilities.TotalLiabilities
	iterateTags(data, currLiabilities, &r.BalanceSheet.Liabilities.CurrentLiabilities)
	iterateTags(data, nonCurrLiabilities, &r.BalanceSheet.Liabilities.NonCurrentLiabilities)
	iterateTags(data, totalLiabilities, &r.BalanceSheet.Liabilities.TotalLiabilities)
	// Assemble equity
	equity := xbrl.Tags.BalanceSheetItems.Equity
	totalLiabilitiesEquity := xbrl.Tags.BalanceSheetItems.TotalLiabilitiesAndEquity
	iterateTags(data, equity, &r.BalanceSheet.Equity)
	iterateTags(data, totalLiabilitiesEquity, &r.BalanceSheet.TotalLiabilitiesAndEquity)
}

func assembleIncomeStatement(f *types.CompanyFacts, xbrl *types.XBRLTags, r *types.FinancialStatement) {
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
	iterateTags(data, revenue, &r.IncomeStatement.Revenue)
	iterateTags(data, cogs, &r.IncomeStatement.CostOfRevenue)
	iterateTags(data, gp, &r.IncomeStatement.GrossProfit)
	iterateTags(data, opex, &r.IncomeStatement.OperatingExpenses)
	iterateTags(data, opIncomeLos, &r.IncomeStatement.OperatingIncomeLoss)
	iterateTags(data, other, &r.IncomeStatement.OtherIncomeExpense)
	iterateTags(data, incomeBeforeTax, &r.IncomeStatement.IncomeBeforeTax)
	iterateTags(data, tax, &r.IncomeStatement.IncomeBeforeTax)
	iterateTags(data, ni, &r.IncomeStatement.NetIncomeLoss)
}

func assembleCashFlowStatement(f *types.CompanyFacts, xbrl *types.XBRLTags, r *types.FinancialStatement) {
	data := f.Facts.Data
	opActivities := xbrl.Tags.CashFlowStatementItems.OperatingActivities
	investingActivities := xbrl.Tags.CashFlowStatementItems.InvestingActivities
	financingActivities := xbrl.Tags.CashFlowStatementItems.FinancingActivities
	// TODO: ?
	cash := xbrl.Tags.CashFlowStatementItems.CashAndCashEquivalents
	iterateTags(data, opActivities, &r.CashFlowStatement.OperatingActivities)
	iterateTags(data, investingActivities, &r.CashFlowStatement.InvestingActivities)
	iterateTags(data, financingActivities, &r.CashFlowStatement.FinancingActivities)
	iterateTags(data, cash, &r.CashFlowStatement.CashAndCashEquivalents)
}

func iterateTags(d map[string]types.FactData, item []string, l *[]types.LineItem) {
	for i := 0; i < len(item); i++ {
		factData, ok := d[item[i]]
		if ok {
			newLi := &types.LineItem{Tag: factData.Label, Data: factData.Units.USD}
			*l = append(*l, *newLi)
		}
	}
}

// Adds leading zeroes to a CIK string to make it 10 characters long and compatible
// with SEC API.
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
