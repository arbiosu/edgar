package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/arbiosu/edgar/types"
	"golang.org/x/net/html"
)

func setupFlags(s *types.ServerConfig, g *types.GetConfig, p *types.ParseConfig) map[string]*flag.FlagSet {
	sh := "(shorthand)"
	now := time.Now()
	year := now.Year()

	server := flag.NewFlagSet("server", flag.ExitOnError)
	server.IntVar(&s.Port, "port", 8080, "Server port")
	server.StringVar(&s.Email, "email", "hello@example.com", "Your email address")
	server.StringVar(&s.Usage, "usage", "personal use", "Usage statement")

	get := flag.NewFlagSet("get", flag.ExitOnError)
	get.StringVar(&g.CIK, "cik", "", "CIK number")
	get.StringVar(&g.Ticker, "ticker", "", "Stock ticker")
	get.StringVar(&g.Ticker, "t", "", "Stock ticker"+sh)
	get.StringVar(&g.Doc, "doc", "10-K", "Desired document (10-k, 10-Q)")
	get.StringVar(&g.Doc, "d", "10-K", "Desired document"+sh)
	get.IntVar(&g.Period, "period", year, "Time period")
	get.IntVar(&g.Period, "p", year, "Time period"+sh)
	// TODO: fix or remove?
	get.StringVar(&g.RawFile, "save", "edgar.html", "Name of the raw file to be parsed")
	get.StringVar(&g.RawFile, "s", "edgar.html", "Name of the raw file to be parsed"+sh)

	// TODO: fix
	parse := flag.NewFlagSet("parse", flag.ExitOnError)
	parse.StringVar(&p.File, "file", "edgar", "Name of the raw file to be parsed")
	parse.StringVar(&p.Format, "format", "html", "Format of the parsed file")
	parse.StringVar(&p.Format, "f", "html", "Format of the parsed file")
	parse.StringVar(&p.Output, "output", p.File+"."+p.Format, "Name of the output file")
	parse.StringVar(&p.Output, "o", p.File+"."+p.Format, "Name of the output file")

	m := make(map[string]*flag.FlagSet)
	m["server"] = server
	m["get"] = get
	m["parse"] = parse

	return m
}

func main() {

	// TODO: check for previous server config and company_tickers.json

	var serverConfig types.ServerConfig
	var getConfig types.GetConfig
	var parseConfig types.ParseConfig
	m := setupFlags(&serverConfig, &getConfig, &parseConfig)

	if len(os.Args) < 2 {
		fmt.Println("expected 'server' or 'get' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		m["server"].Parse(os.Args[2:])
		handleServer(serverConfig)
		fmt.Printf("EDGAR Server Configuration: %+v\n", serverConfig)
	case "get":
		m["get"].Parse(os.Args[2:])
		handleGet(&getConfig)
		fmt.Printf("%+v\n", getConfig)
	case "parse":
		m["parse"].Parse(os.Args[2:])
		fmt.Printf("%+v\n", parseConfig)
	default:
		fmt.Println("Expected 'server' or 'get' subcommands")
		os.Exit(1)
	}
}

func handleServer(config types.ServerConfig) {
	err := createDir("config")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	bytes, err := json.Marshal(config)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = os.WriteFile("config/config.json", bytes, 0660)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func handleGet(getConfig *types.GetConfig) {
	/*
		serverConfig := checkConfig()
		// TODO: handle when CIK is given
		tickers := checkCompanyTickers(serverConfig)
		cik, ok := tickers[getConfig.Ticker]
		if !ok {
			// TODO: handle err
			fmt.Printf("Error: Ticker not found! Exiting program.\n")
			os.Exit(1)
		}
		cikStr := strconv.Itoa(cik)
		url := assembleSubmissionsUrl(cikStr, getConfig)
		urls := getFileUrls(url, serverConfig, getConfig)
		err := downloadFiles(urls, serverConfig, getConfig)
		if err != nil {
			fmt.Printf("Error: Failed to download files! (%v)\n", err)
			os.Exit(1)
		}
	*/
	d, err := parseFile(getConfig)
	if err != nil {
		os.Exit(1)
	}
	for i := 0; i < 100; i++ {
		fmt.Printf("Category: %v\nYear1: %v\n Y2: %v\n Y3: %v\n", d[i].Category, d[i].Year1, d[i].Year2, d[i].Year3)
	}
}

// Creates a directory
func createDir(name string) error {
	var err error
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

func checkConfig() *types.ServerConfig {
	fmt.Println("Checking for previous server configuration...")
	c, err := os.ReadFile("./config/config.json")
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}
	var serverConfig types.ServerConfig
	err = json.Unmarshal(c, &serverConfig)

	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Config found!")
	return &serverConfig
}

// Makes a GET request to the given URL and returns the response body.
func makeSecRequest(url string, s *types.ServerConfig) ([]byte, error) {
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = http.Header{
		"User-Agent":   {s.Usage + " " + s.Email},
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
func getFileUrls(url string, s *types.ServerConfig, g *types.GetConfig) []string {
	body, err := makeSecRequest(url, s)
	if err != nil {
		fmt.Printf("Error: could not make SEC Request! (%v)\n", err)
		// TODO: handle
	}
	var c types.Company
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
	for i, v := range c.Filings.Recent.Form {
		// TODO: Validate g.Period with c.Filings.Recent.FilingDate[i]
		// Ensure g.Doc is correct ie: 10-K, 10-Q
		if v == g.Doc {
			// strip '-' from accession number
			re := regexp.MustCompile(`-`)
			cleaned := re.ReplaceAllString(c.Filings.Recent.AccessionNumber[i], "")
			urls = append(urls, newUrl+c.Cik+"/"+cleaned+"/"+c.Filings.Recent.PrimaryDocument[i])

		}
	}
	return urls
}

func downloadFiles(urls []string, s *types.ServerConfig, g *types.GetConfig) error {
	err := createDir("app/" + g.Ticker + "/")
	if err != nil {
		fmt.Printf("Error: could not create 'app' directory! (%v)\n", err)
		os.Exit(1)
		// TODO: handle
	}
	for i := 0; i < len(urls); i++ {
		body, err := makeSecRequest(urls[i], s)
		if err != nil {
			fmt.Printf("Error: could not request %s! (%v)\n", urls[i], err)
			// TODO: Handle
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

func parseFile(g *types.GetConfig) ([]types.FinancialData, error) {
	file, err := os.Open(g.RawFile)
	if err != nil {
		fmt.Printf("Error: could not parse html file!\n", err)
		os.Exit(1)
		// TODO: handle
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	doc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}
	var financialData []types.FinancialData
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			data := extractRowData(n)
			if data.Category != "" {
				financialData = append(financialData, data)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return financialData, nil
}

func extractRowData(n *html.Node) types.FinancialData {
	var data types.FinancialData
	var cellIdx int

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			text := extractText(c)
			switch cellIdx {
			case 0:
				data.Category = strings.TrimSpace(text)
			case 3:
				data.Year1 = strings.TrimSpace(text)
			case 7:
				data.Year2 = strings.TrimSpace(text)
			case 11:
				data.Year3 = strings.TrimSpace(text)
			}
			cellIdx++
		}
	}
	return data
}

func extractText(n *html.Node) string {
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			text += c.Data
		}
		text += extractText(c)
	}
	return text
}

// Downloads company_tickers.json to config/ directory. Returns an error if
// unsuccessful
func getCompanyTickers(s *types.ServerConfig) error {
	// TODO: replace with makeSecRequest
	// make url a const
	// pass it in function call to sec request
	// write to config/company_tickers.json
	client := http.Client{}
	req, err := http.NewRequest("GET", "https://www.sec.gov/files/company_tickers.json", nil)
	if err != nil {
		return err
	}
	req.Header = http.Header{
		"User-Agent":   {s.Usage + " " + s.Email},
		"Content-Type": {"application/json"},
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = os.WriteFile("config/company_tickers.json", body, 0666)
	if err != nil {
		return err
	}
	return nil
}

// Check for company_tickers.json file. If it exists, returns map of tickers
// If it does not exist, make a request and download it from SEC
func checkCompanyTickers(s *types.ServerConfig) map[string]int {
	data, err := os.ReadFile("config/company_tickers.json")
	if err != nil {
		fmt.Printf("Error: could not find company_tickers.json! (%v)\n", err)
		// TODO: handle
		fmt.Println("Requesting file...")
		err = getCompanyTickers(s)
		if err != nil {
			fmt.Printf("Error: could not get company_tickers.json! (%v)\n", err)
			fmt.Println("Exiting program...")
			os.Exit(1)
		} else {
			fmt.Println("Success! File company_tickers.json downloaded!")
			// Recursive call to unmarshal JSON
			// TODO: Bug here. File downloads then runs into unexpected end of
			// JSON input. May change to return here and instruct to try again
			checkCompanyTickers(s)
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
// TODO: fix this description
func editCompanyTickers(tickers map[int]types.Ticker) map[string]int {
	m := make(map[string]int)
	for _, v := range tickers {
		m[v.Tick] = v.Cik
	}
	return m
}
*/

// Adds leading zeroes to a CIK string to make it 10 characters long.
func zeroPad(cik string) string {
	for len(cik) < 10 {
		cik = "0" + cik
	}
	cik = "CIK" + cik
	return cik
}

// Assemble the submissions url to get company info. Update the GetConfig struct.
func assembleSubmissionsUrl(cik string, g *types.GetConfig) string {
	cik = zeroPad(cik)
	g.CIK = cik
	return "https://data.sec.gov/submissions/" + cik + ".json"
}
