package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func (a *App) ShowCurrentState() error {
	isin, err := a.DB().GetAllISIN()
	if err != nil {
		return err
	}

	sort.Slice(isin, func(i, j int) bool {
		return isin[i].ID < isin[j].ID
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ISIN", "Name", "Last update", "Value per share", "Shares", "Owned value"})
	table.SetRowLine(true)

	totals := map[string]float64{}

	for _, i := range isin {
		vps, err := a.currency.Localize(i.Nomination, i.ValuePerShare)
		if err != nil {
			a.Logger().Error(err)
		}

		ov, err := a.currency.Localize(i.Nomination, i.OwnedValue())
		if err != nil {
			a.Logger().Error(err)
		}

		totals[i.Nomination] += i.OwnedValue()

		table.Append([]string{
			i.ID, i.Name, timeToDate(&i.UpdatedAt), vps, fmt.Sprintf("%.2f", i.Shares), ov,
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

	table.Append([]string{"", "", "", "", "", strings.Join(tvs, "\n")})
	table.Render()

	return nil
}
