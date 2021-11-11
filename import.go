package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/asdine/storm/v3"
)

var (
	valuationMatch = regexp.MustCompile(
		`<tr><td class="mod-ui-table__cell--text">` +
			`<span class="mod-ui-hide-small-below">(?P<longdate>[^<]+)</span>` +
			`<span class="mod-ui-hide-medium-above">(?P<shortdate>[^<]+)</span></td>` +
			`<td>(?P<open>[0-9\.,]+)</td>` +
			`<td>(?P<high>[0-9\.,]+)</td>` +
			`<td>(?P<low>[0-9\.,]+)</td>` +
			`<td>(?P<close>[0-9\.,]+)</td>`)
	ErrParseFTInput = errors.New("could not parse FT input")
)

func parseFTInput(isin string, data string) ([]*Valuation, error) {
	var res []*Valuation

	groupNames := valuationMatch.SubexpNames()

	for _, match := range valuationMatch.FindAllStringSubmatch(data, -1) {
		result := map[string]string{}

		for groupIdx, group := range match {
			name := groupNames[groupIdx]
			if name == "" {
				continue
			}

			result[name] = group
		}

		parsed, err := mapToValuation(isin, result)
		if err != nil {
			return nil, err
		}

		res = append(res, parsed)
	}

	return res, nil
}

func mapToValuation(isin string, in map[string]string) (*Valuation, error) {
	d, err := time.Parse("Mon, Jan 02, 2006", in["shortdate"])
	if err != nil {
		return nil, fmt.Errorf("%w: shortdate", ErrParseFTInput)
	}

	o, err := strToF64(in["open"])
	if err != nil {
		return nil, fmt.Errorf("%w: open", ErrParseFTInput)
	}

	h, err := strToF64(in["high"])
	if err != nil {
		return nil, fmt.Errorf("%w: high", ErrParseFTInput)
	}

	l, err := strToF64(in["low"])
	if err != nil {
		return nil, fmt.Errorf("%w: low", ErrParseFTInput)
	}

	c, err := strToF64(in["close"])
	if err != nil {
		return nil, fmt.Errorf("%w: close", ErrParseFTInput)
	}

	v := Valuation{ISIN: isin, Date: d, Open: o, High: h, Low: l, Close: c}

	return &v, nil
}

func strToF64(in string) (float64, error) {
	in = strings.ReplaceAll(in, ",", "")

	return strconv.ParseFloat(in, 32)
}

func (db *DB) ImportValuations(isin *ISIN, valuations []*Valuation) error {
	var newR Valuation

	sort.Slice(valuations, func(i, j int) bool {
		return valuations[i].Date.After(valuations[j].Date)
	})

	for _, v := range valuations {
		v.UpdateID()
		if isin.UpdatedAt.Before(v.Date) {
			isin.ValuePerShare = v.Value()
			isin.UpdatedAt = v.Date

			db.logger.Infof("New value for '%s': %s %.2f (%s)", isin.ID, isin.Nomination, isin.ValuePerShare, isin.UpdatedAt.UTC())

			if err := db.DB().Save(isin); err != nil {
				return err
			}
		}

		err := db.DB().One("ID", v.ID, &newR)
		if err == nil {
			// We have it already...
			continue
		}

		if !errors.Is(err, storm.ErrNotFound) {
			return err
		}

		db.logger.Debugf("Adding entry: %#v", v.ID)
		if err := db.DB().Save(v); err != nil {
			return err
		}
	}

	return nil
}
