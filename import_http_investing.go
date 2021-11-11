package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strconv"
	"time"

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

var (
	extractInvestingNomination = regexp.MustCompile(`<span class="bold pid-\d+-time">.* Valuta in <span class='bold'>([^<]+)</span>`)
)

func (db *DB) InvestingUpdateFromHTTP(i *ISIN) error {
	c := retryablehttp.NewClient()
	c.Logger = db.logger

	if err := db.InvestingUpdateMetaFromHTTP(i, c); err != nil {
		return err
	}

	if i.XID == "" {
		return nil
	}

	return db.InvestingUpdateValuationsFromHTTP(i, c)
}

func (db *DB) InvestingUpdateMetaFromHTTP(i *ISIN, c *retryablehttp.Client) error {
	u, err := url.Parse("https://nl.investing.com/search/service/searchTopBar")
	if err != nil {
		return err
	}

	b := url.Values{
		"search_text": []string{i.ID},
	}

	req, err := retryablehttp.NewRequest("POST", u.String(), []byte(b.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Me")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Do(req)
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

	i.XID = strconv.Itoa(q.PairID)
	i.Name = q.Name
	i.AssetClass = q.PairType

	return db.DB().Save(i)
}

func (db *DB) InvestingUpdateValuationsFromHTTP(i *ISIN, c *retryablehttp.Client) error {
	curTS := fmt.Sprintf("%d", time.Now().Unix())

	u, err := url.Parse("https://tvc4.investing.com/1d34c13b0d6656b98005c7e69f95ccf7/" + curTS + "/36/16/16/history")
	if err != nil {
		return err
	}

	q := url.Values{
		"symbol": []string{i.XID},
		"from":   []string{"1000000000"},
		"to":     []string{curTS},
	}

	u.RawQuery = q.Encode()

	req, err := retryablehttp.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Me")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var output InvestingSeries

	if err := json.Unmarshal(body, &output); err != nil {
		return err
	}

	vals, err := investingToValuaions(i.ID, output)
	if err != nil {
		return err
	}

	db.logger.Debugf("got %d valuations", len(vals))

	return db.ImportValuations(i, vals)
}

func investingToValuaions(isin string, s InvestingSeries) ([]*Valuation, error) {
	if s.Status != "ok" {
		return nil, nil
	}

	result := make([]*Valuation, len(s.Timestamps))

	for i, d := range s.Timestamps {
		parsed := time.Unix(d, 0)

		result[i] = &Valuation{
			ISIN:  isin,
			Date:  parsed,
			Open:  s.Opens[i],
			High:  s.Highs[i],
			Low:   s.Lows[i],
			Close: s.Closes[i],
		}
	}

	return result, nil
}
