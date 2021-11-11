package main

import (
	"errors"
	"fmt"
)

const (
	DataSourceFT        = "FT"
	DataSourceInvesting = "investing"
)

var (
	ErrUnknownSource = errors.New("unknown data source")
)

func (db *DB) UpdateFromHTTP(i *ISIN) error {
	switch i.Source {
	case DataSourceFT:
		return db.FTUpdateFromHTTP(i)
	case DataSourceInvesting:
		return db.InvestingUpdateFromHTTP(i)
	default:
		return fmt.Errorf("%w: '%s'", ErrUnknownSource, i.Source)
	}
}
