package main

import (
	"encoding/json"
	"html"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type FTSeries struct {
	Dates        []string          `json:"Dates"`
	Status       int               `json:"Status"`
	StatusString string            `json:"StatusString"`
	Elements     []FTSeriesElement `json:"Elements"`
}

type FTSeriesElement struct {
	CompanyName     string              `json:"CompanyName"`
	Symbol          string              `json:"Symbol"`
	Currency        string              `json:"Currency"`
	ComponentSeries []FTSeriesComponent `json:"ComponentSeries"`
}

type FTSeriesComponent struct {
	Type         string    `json:"Type"`
	MaxValue     float64   `json:"MaxValue"`
	MinValue     float64   `json:"MinValue"`
	MaxValueDate string    `json:"MaxValueDate"`
	MinValueDate string    `json:"MinValueDate"`
	Values       []float64 `json:"Values"`
}

type FTSeriesQuery struct {
	Days              int                    `json:"days"`
	DataNormalized    bool                   `json:"dataNormalized"`
	DataPeriod        string                 `json:"dataPeriod"`
	DataInterval      int                    `json:"dataInterval"`
	Realtime          bool                   `json:"realtime"`
	TimeServiceFormat string                 `json:"timeServiceFormat"`
	ReturnDateType    string                 `json:"returnDateType"`
	Elements          []FTSeriesQueryElement `json:"elements"`
}

type FTSeriesQueryElement struct {
	Label  string `json:"Label"`
	Type   string `json:"Type"`
	Symbol string `json:"Symbol"`
}

type FTData struct {
	XID    string `json:"xid"`
	Symbol string `json:"symbol"`
}

type FTSearch struct {
	Data struct {
		Security []struct {
			Name       string `json:"name"`
			Symbol     string `json:"symbol"`
			AssetClass string `json:"assetClass"`
		} `json:"security"`
	} `json:"data"`
}

var extractFTInfo = regexp.MustCompile(`.*<section class="mod-tearsheet-add-to-watchlist" data-mod-config="([^"]+)".*`)

func (i *ISIN) BuildFTSeriesQuery(days int) FTSeriesQuery {
	return FTSeriesQuery{
		Days:              days,
		DataPeriod:        "Day",
		DataInterval:      1,
		TimeServiceFormat: "JSON",
		ReturnDateType:    "ISO8601",
		Elements: []FTSeriesQueryElement{
			{
				Label:  "3ec7c513",
				Type:   "price",
				Symbol: i.XID,
			},
		},
	}
}

func (db *DB) FTUpdateFromHTTP(isin *ISIN) error {
	client := retryablehttp.NewClient()
	client.Logger = db.logger

	if err := db.FTUpdateXIDFromHTTP(isin, client); err != nil {
		return err
	}

	if err := db.FTUpdateMetaFromHTTP(isin, client); err != nil {
		return err
	}

	return db.FTUpdateValuationsFromHTTP(isin, client)
}

func (db *DB) FTUpdateXIDFromHTTP(isin *ISIN, client *retryablehttp.Client) error {
	u, err := url.Parse("https://markets.ft.com/data/funds/tearsheet/charts?s=" + isin.ISINNomination())
	if err != nil {
		return err
	}

	req, err := retryablehttp.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	parsed := extractFTInfo.FindStringSubmatch(string(body))
	if len(parsed) < 2 {
		return nil
	}

	j := html.UnescapeString(parsed[1])

	var d FTData

	if err := json.Unmarshal([]byte(j), &d); err != nil {
		return err
	}

	isin.XID = d.XID

	return db.DB().Save(isin)
}

func (db *DB) FTUpdateMetaFromHTTP(isin *ISIN, client *retryablehttp.Client) error {
	u, err := url.Parse("https://markets.ft.com/data/searchapi/searchsecurities?query=" + isin.ISINNomination())
	if err != nil {
		return err
	}

	req, err := retryablehttp.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var output FTSearch

	if err := json.Unmarshal(body, &output); err != nil {
		return err
	}

	if len(output.Data.Security) == 0 {
		return nil
	}

	r0 := output.Data.Security[0]
	symb := strings.SplitN(r0.Symbol, ":", 2)

	isin.Name = r0.Name
	isin.Nomination = symb[1]
	isin.AssetClass = r0.AssetClass

	return db.DB().Save(isin)
}

func (db *DB) FTUpdateValuationsFromHTTP(isin *ISIN, client *retryablehttp.Client) error {
	ftURL, err := url.Parse("https://markets.ft.com/data/chartapi/series")
	if err != nil {
		return err
	}

	query := isin.BuildFTSeriesQuery(1000)

	j, err := json.Marshal(&query)
	if err != nil {
		return err
	}

	req, err := retryablehttp.NewRequest("POST", ftURL.String(), j)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var output FTSeries

	if err = json.Unmarshal(body, &output); err != nil {
		return err
	}

	vals, err := ftToValuaions(isin.ID, output)
	if err != nil {
		return err
	}

	return db.ImportValuations(isin, vals)
}

func ftToValuaions(isin string, series FTSeries) ([]*Valuation, error) {
	result := make([]*Valuation, len(series.Dates))

	if len(series.Elements) == 0 {
		return nil, nil
	}

	e := series.Elements[0]

	open := ftGetComponent(e, "Open")
	high := ftGetComponent(e, "High")
	low := ftGetComponent(e, "Low")
	closeValues := ftGetComponent(e, "Close")

	for seq, date := range series.Dates {
		parsed, err := time.Parse("2006-01-02T15:04:05", date)
		if err != nil {
			return nil, err
		}

		result[seq] = &Valuation{
			ISIN:  isin,
			Date:  parsed,
			Open:  open.Values[seq],
			High:  high.Values[seq],
			Low:   low.Values[seq],
			Close: closeValues.Values[seq],
		}
	}

	return result, nil
}

func ftGetComponent(e FTSeriesElement, n string) *FTSeriesComponent {
	for _, c := range e.ComponentSeries {
		if c.Type == n {
			return &c
		}
	}

	return nil
}
