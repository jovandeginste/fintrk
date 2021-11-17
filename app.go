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

var (
	singeStateHeaders = []string{"ISIN", "Name", "Nom", "Last update", "Value per share", "Shares", "Owned value"}
	sinceStateHeaders = []string{"ISIN", "Name", "Nom", "Last update", "Previous value", "Current value", "Change"}
)

func (a *App) ShowStateSince(tableFormat string, date time.Time) error {
	isins, err := a.DB().GetAllISIN()
	if err != nil {
		return err
	}

	sort.Slice(isins, func(i, j int) bool {
		return isins[i].ID < isins[j].ID
	})

	var entries [][]string

	totals1 := map[string]float64{}
	totals2 := map[string]float64{}

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
		totals1[isin.Nomination] += ownedValue
		totals2[isin.Nomination] += isin.OwnedValue()
		diff := isin.OwnedValue() - ownedValue

		entries = append(entries, a.buildSinceTableEntry(
			isin.ID, isin.Name, &valuation.Date, isin.Nomination, ownedValue, isin.OwnedValue(), diff,
		))
	}

	a.showSinceTable(tableFormat, entries, totals1, totals2)

	return nil
}

func (a *App) ShowStateAt(tableFormat string, date time.Time) error {
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

		entries = append(entries, a.buildSingleTableEntry(
			isin.ID, isin.Name, &valuation.Date, isin.Nomination, valuation.Value(), shares, ownedValue,
		))
	}

	a.showSingleTable(tableFormat, entries, totals)

	return nil
}

func (a *App) ShowCurrentState(tableFormat string) error {
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

		entries = append(entries, a.buildSingleTableEntry(
			isin.ID, isin.Name, &isin.UpdatedAt, isin.Nomination, isin.ValuePerShare, isin.Shares, isin.OwnedValue(),
		))
	}

	a.showSingleTable(tableFormat, entries, totals)

	return nil
}

func (a *App) buildSinceTableEntry(isinID string, isinName string, date *time.Time, nomination string, val1, val2, diff float64) []string {
	locVal1, err := a.currency.Localize(nomination, val1)
	if err != nil {
		a.Logger().Error(err)
	}

	locVal2, err := a.currency.Localize(nomination, val2)
	if err != nil {
		a.Logger().Error(err)
	}

	locDiff, err := a.currency.Localize(nomination, diff)
	if err != nil {
		a.Logger().Error(err)
	}

	return []string{
		isinID, isinName, nomination, timeToDate(date), locVal1, locVal2, locDiff,
	}
}

func (a *App) buildSingleTableEntry(isinID string, isinName string, date *time.Time, nomination string, valuePerShare float64, shares float64, ownedValue float64) []string {
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

func (a *App) showSingleTable(tableFormat string, entries [][]string, totals map[string]float64) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(singeStateHeaders)
	configureRenderer(table, tableFormat)

	table.AppendBulk(entries)

	for nom, value := range totals {
		tv, err := a.currency.Localize(nom, value)
		if err != nil {
			a.Logger().Error(err)
		}

		table.Append([]string{"Total", "", nom, "", "", "", tv})
	}

	table.Render()
}

func (a *App) showSinceTable(tableFormat string, entries [][]string, totals1, totals2 map[string]float64) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(sinceStateHeaders)
	configureRenderer(table, tableFormat)

	table.AppendBulk(entries)

	for nom, value2 := range totals2 {
		value1 := totals1[nom]
		diff := value2 - value1

		locVal1, err := a.currency.Localize(nom, value1)
		if err != nil {
			a.Logger().Error(err)
		}

		locVal2, err := a.currency.Localize(nom, value2)
		if err != nil {
			a.Logger().Error(err)
		}

		tvDiff, err := a.currency.Localize(nom, diff)
		if err != nil {
			a.Logger().Error(err)
		}

		table.Append([]string{"Total", "", nom, "", locVal1, locVal2, tvDiff})
	}

	table.Render()
}

func configureRenderer(table *tablewriter.Table, tableFormat string) {
	switch tableFormat {
	case "markdown", "md":
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetAutoWrapText(false)
	default:
		table.SetRowLine(true)
	}
}
