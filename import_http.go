package main

import (
	"errors"
	"fmt"
)

const (
	DataSourceFT        = "FT"
	DataSourceInvesting = "investing"
)

var ErrUnknownSource = errors.New("unknown data source")

func (db *DB) UpdateFromHTTP(isin *ISIN) error {
	switch isin.Source {
	case DataSourceFT:
		return db.FTUpdateFromHTTP(isin)
	case DataSourceInvesting:
		return db.InvestingUpdateFromHTTP(isin)
	default:
		return fmt.Errorf("%w: '%s'", ErrUnknownSource, isin.Source)
	}
}
