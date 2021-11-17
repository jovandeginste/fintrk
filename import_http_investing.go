package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/hashicorp/go-retryablehttp"
)

type InvestingSearchResponse struct {
	Total struct {
		AllResults int `json:"allResults"`
		Quotes     int `json:"quotes"`
	} `json:"total"`
	Quotes []InvestingSearchResponseQuote `json:"quotes"`
}

type InvestingSearchResponseQuote struct {
	PairID      int    `json:"pairId"`
	Name        string `json:"name"`
	Flag        string `json:"flag"`
	Link        string `json:"link"`
	Symbol      string `json:"symbol"`
	Type        string `json:"type"`
	PairTypeRaw string `json:"pair_type_raw"`
	PairType    string `json:"pair_type"`
	CountryID   int    `json:"countryID"`
	Sector      int    `json:"sector"`
	Region      int    `json:"region"`
	Industry    int    `json:"industry"`
	IsCrypto    bool   `json:"isCrypto"`
	Exchange    string `json:"exchange"`
	ExchangeID  int    `json:"exchangeID"`
}

type InvestingSeries struct {
	Timestamps []int64   `json:"t"`
	Closes     []float64 `json:"c"`
	Opens      []float64 `json:"o"`
	Highs      []float64 `json:"h"`
	Lows       []float64 `json:"l"`
	Status     string    `json:"s"`
}

func (db *DB) InvestingUpdateFromHTTP(isin *ISIN) error {
	client := retryablehttp.NewClient()
	client.Logger = db.logger

	if err := db.InvestingUpdateMetaFromHTTP(isin, client); err != nil {
		return err
	}

	if isin.XID == "" {
		return nil
	}

	return db.InvestingUpdateValuationsFromHTTP(isin, client)
}

func (db *DB) InvestingUpdateMetaFromHTTP(isin *ISIN, client *retryablehttp.Client) error {
	invURL, err := url.Parse("https://nl.investing.com/search/service/searchTopBar")
	if err != nil {
		return err
	}

	b := url.Values{
		"search_text": []string{isin.ID},
	}

	db.logger.Debugf("Body: %s", b.Encode())

	req, err := retryablehttp.NewRequest("POST", invURL.String(), []byte(b.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Me")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var output InvestingSearchResponse

	if err := json.Unmarshal(body, &output); err != nil {
		return err
	}

	if len(output.Quotes) == 0 {
		return nil
	}

	q := output.Quotes[0]

	isin.XID = strconv.Itoa(q.PairID)
	isin.Name = q.Name
	isin.AssetClass = q.PairType

	return db.DB().Save(isin)
}

func (db *DB) InvestingUpdateValuationsFromHTTP(isin *ISIN, client *retryablehttp.Client) error {
	sinceTS := "1000000000"
	curTS := fmt.Sprintf("%d", time.Now().Unix())

	v, err := db.GetValuation(isin.ID)
	if err != nil {
		if !errors.Is(err, storm.ErrNotFound) {
			return err
		}
	} else {
		sinceTS = fmt.Sprintf("%d", v.Date.Unix())
	}

	invURL, err := url.Parse("https://tvc4.investing.com/1d34c13b0d6656b98005c7e69f95ccf7/" + curTS + "/36/16/16/history")
	if err != nil {
		return err
	}

	query := url.Values{
		"symbol": []string{isin.XID},
		"from":   []string{sinceTS},
		"to":     []string{curTS},
	}

	invURL.RawQuery = query.Encode()

	req, err := retryablehttp.NewRequest("GET", invURL.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Me")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var output InvestingSeries

	if err = json.Unmarshal(body, &output); err != nil {
		return err
	}

	vals, err := investingToValuaions(isin.ID, output)
	if err != nil {
		return err
	}

	db.logger.Debugf("got %d valuations", len(vals))

	return db.ImportValuations(isin, vals)
}

func investingToValuaions(isin string, series InvestingSeries) ([]*Valuation, error) { //nolint:unparam
	if series.Status != "ok" {
		return nil, nil
	}

	result := make([]*Valuation, len(series.Timestamps))

	for seq, d := range series.Timestamps {
		parsed := time.Unix(d, 0)

		result[seq] = &Valuation{
			ISIN:  isin,
			Date:  parsed,
			Open:  series.Opens[seq],
			High:  series.Highs[seq],
			Low:   series.Lows[seq],
			Close: series.Closes[seq],
		}
	}

	return result, nil
}
