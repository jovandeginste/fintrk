package main

import (
	"errors"
	"sort"

	"github.com/asdine/storm/v3"
)

func (db *DB) ImportValuations(isin *ISIN, valuations []*Valuation) error {
	var newR Valuation

	sort.Slice(valuations, func(i, j int) bool {
		return valuations[i].Date.After(valuations[j].Date)
	})

	for _, val := range valuations {
		val.UpdateID()

		if isin.UpdatedAt.Before(val.Date) {
			isin.ValuePerShare = val.Value()
			isin.UpdatedAt = val.Date

			db.logger.Infof("New value for '%s': %s %.2f (%s)", isin.ID, isin.Nomination, isin.ValuePerShare, isin.UpdatedAt.UTC())

			if err := db.DB().Save(isin); err != nil {
				return err
			}
		}

		err := db.DB().One("ID", val.ID, &newR)
		if err == nil {
			// We have it already...
			continue
		}

		if !errors.Is(err, storm.ErrNotFound) {
			return err
		}

		db.logger.Debugf("Adding entry: %#v", val.ID)

		if err := db.DB().Save(val); err != nil {
			return err
		}
	}

	return nil
}
