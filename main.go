package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/arbiosu/edgar/types"
)

func setupFlags(c *types.ClientConfig, g *types.GetConfig) map[string]*flag.FlagSet {

	var (
		sh     = "(shorthand)"
		now    = time.Now()
		year   = now.Year()
		client = flag.NewFlagSet("client", flag.ExitOnError)
		get    = flag.NewFlagSet("get", flag.ExitOnError)
		email  = "Your email address"
		usage  = "Usage statement"
		cik    = "CIK number"
		ticker = "Stock ticker"
		doc    = "Desired document (10-K, 10-Q)"
		period = "Time period"
		save   = "Name of the file to be saved"
		format = "Download raw HTML files or get a JSON report"
	)

	client.StringVar(&c.Email, "email", "hello@example.com", email)
	client.StringVar(&c.Email, "e", "hello@example.com", email+sh)
	client.StringVar(&c.Usage, "usage", "personal use", usage)
	client.StringVar(&c.Usage, "u", "personal use", usage+sh)

	get.StringVar(&g.CIK, "cik", "", cik)
	get.StringVar(&g.Ticker, "ticker", "", ticker)
	get.StringVar(&g.Ticker, "t", "", ticker+sh)
	get.StringVar(&g.Doc, "doc", "10-K", doc)
	get.StringVar(&g.Doc, "d", "10-K", doc+sh)
	get.IntVar(&g.Period, "period", year, period)
	get.IntVar(&g.Period, "p", year, period+sh)
	get.StringVar(&g.RawFile, "save", "", save)
	get.StringVar(&g.RawFile, "s", "", save+sh)
	get.StringVar(&g.Format, "format", "html", format)
	get.StringVar(&g.Format, "f", "html", format+sh)

	m := make(map[string]*flag.FlagSet)
	m["client"] = client
	m["get"] = get

	return m
}

func main() {
	c := &types.ClientConfig{}
	g := &types.GetConfig{}
	m := setupFlags(c, g)

	if len(os.Args) < 2 {
		fmt.Println("Error: expected 'client' or 'get' subcommands. Exiting...")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "client":
		m["client"].Parse(os.Args[2:])
		c.HandleClient()
		fmt.Printf("EDGAR Client Configuration: %+v\n", *c)
	case "get":
		m["get"].Parse(os.Args[2:])
		g.HandleGet()
	case "parse":
		m["parse"].Parse(os.Args[2:])
	default:
		fmt.Println("Expected 'client' or 'get' subcommands")
		os.Exit(1)
	}
}
