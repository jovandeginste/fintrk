package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

func (a *App) ShowStateAt(d time.Time) error {
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

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ISIN", "Name", "Last update", "Value per share", "Shares", "Owned value"})
	table.SetRowLine(true)

	totals := map[string]float64{}

	for _, isin := range isins {
		vps, err := a.currency.Localize(isin.Nomination, isin.ValuePerShare)
		if err != nil {
			a.Logger().Error(err)
		}

		ownedValue, err := a.currency.Localize(isin.Nomination, isin.OwnedValue())
		if err != nil {
			a.Logger().Error(err)
		}

		totals[isin.Nomination] += isin.OwnedValue()

		table.Append([]string{
			isin.ID, isin.Name, timeToDate(&isin.UpdatedAt), vps, fmt.Sprintf("%.2f", isin.Shares), ownedValue,
		})
	}

	tvs := []string{}

	for n, v := range totals {
		tv, err := a.currency.Localize(n, v)
		if err != nil {
			a.Logger().Error(err)
		}

		tvs = append(tvs, tv)
	}

	sort.Strings(tvs)

	table.Append([]string{"Total", "", "", "", "", strings.Join(tvs, "\n")})
	table.Render()

	return nil
}
