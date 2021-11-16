package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/olekukonko/tablewriter"
)

func (a *App) ShowStateAt(date time.Time) error {
	isins, err := a.DB().GetAllISIN()
	if err != nil {
		return err
	}

	sort.Slice(isins, func(i, j int) bool {
		return isins[i].ID < isins[j].ID
	})

	var entries [][]string

	totals := map[string]float64{}

	for _, isin := range isins {
		valuation, err := a.DB().GetValuationAt(isin.ID, date)
		if err != nil {
			if !errors.Is(err, storm.ErrNotFound) {
				a.Logger().Error(err)
			}

			continue
		}

		shares, err := a.DB().GetSharesAt(isin.ID, date)
		if err != nil && !errors.Is(err, storm.ErrNotFound) {
			a.Logger().Error(err)
			continue
		}

		ownedValue := valuation.Value() * shares
		totals[isin.Nomination] += ownedValue

		entries = append(entries, a.buildTableEntry(
			isin.ID, isin.Name, &valuation.Date, isin.Nomination, valuation.Value(), shares, ownedValue,
		))
	}

	a.showTable(entries, totals)

	return nil
}

func (a *App) ShowCurrentState() error {
	isins, err := a.DB().GetAllISIN()
	if err != nil {
		return err
	}

	sort.Slice(isins, func(i, j int) bool {
		return isins[i].ID < isins[j].ID
	})

	var entries [][]string

	totals := map[string]float64{}

	for _, isin := range isins {
		totals[isin.Nomination] += isin.OwnedValue()

		entries = append(entries, a.buildTableEntry(
			isin.ID, isin.Name, &isin.UpdatedAt, isin.Nomination, isin.ValuePerShare, isin.Shares, isin.OwnedValue(),
		))
	}

	a.showTable(entries, totals)

	return nil
}

func (a *App) buildTableEntry(isinID string, isinName string, date *time.Time, nomination string, valuePerShare float64, shares float64, ownedValue float64) []string {
	vps, err := a.currency.Localize(nomination, valuePerShare)
	if err != nil {
		a.Logger().Error(err)
	}

	locOwnedValue, err := a.currency.Localize(nomination, ownedValue)
	if err != nil {
		a.Logger().Error(err)
	}

	return []string{
		isinID, isinName, nomination, timeToDate(date), vps, fmt.Sprintf("%.2f", shares), locOwnedValue,
	}
}

func (a *App) showTable(entries [][]string, totals map[string]float64) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ISIN", "Name", "Nom", "Last update", "Value per share", "Shares", "Owned value"})
	table.SetRowLine(true)

	table.AppendBulk(entries)

	for idx, value := range totals {
		tv, err := a.currency.Localize(idx, value)
		if err != nil {
			a.Logger().Error(err)
		}

		table.Append([]string{"Total", "", idx, "", "", "", tv})
	}

	table.Render()
}
