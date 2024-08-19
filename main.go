package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/arbiosu/edgar/types"
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
	err := createConfigDir()
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

func handleGet(get *types.GetConfig) {
	serverConfig := checkConfig()
	// TODO: handle when CIK is given
	tickers := checkCompanyTickers(serverConfig)
	cik, ok := tickers[get.Ticker]
	if !ok {
		// TODO: handle err
		fmt.Printf("Error: Ticker not found! Exiting program.\n")
		os.Exit(1)
	}
	fmt.Printf("CIK: %d\n", cik)
	cikStr := strconv.Itoa(cik)
	url := assembleSubmissionsUrl(cikStr, get)
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("CIK: %s\n", get.CIK)
}

// Creates the config/ directory. Returns nil on success.
func createConfigDir() error {
	err := os.Mkdir("config", 0750)
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

// Downloads company_tickers.json to config/ directory. Returns an error if
// unsuccessful
func getCompanyTickers(s *types.ServerConfig) error {
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
