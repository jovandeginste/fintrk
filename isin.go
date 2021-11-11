package main

import (
	"errors"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
)

type ISIN struct {
	ID         string `storm:"id"`
	XID        string `storm:"index"`
	Name       string
	AssetClass string
	Nomination string
	Source     string

	Shares        float64
	ValuePerShare float64
	UpdatedAt     time.Time

	Valuations   []*Valuation   `json:"-"`
	Transactions []*Transaction `json:"-"`
}

func (i *ISIN) OwnedValue() float64 {
	return i.Shares * i.ValuePerShare
}

func (db *DB) GetISIN(isin string) (*ISIN, error) {
	var i ISIN

	query := db.DB().Select(
		q.Eq("ID", isin),
	).Limit(1)

	if err := query.First(&i); err != nil {
		return nil, err
	}

	return &i, nil
}

func (db *DB) GetISINByXID(xid string) (*ISIN, error) {
	var i ISIN

	query := db.DB().Select(
		q.Eq("XID", xid),
	).Limit(1)

	if err := query.First(&i); err != nil {
		return nil, err
	}

	return &i, nil
}

func (i *ISIN) ISINNomination() string {
	if i.Nomination == "" {
		return i.ID
	}

	return i.ID + ":" + i.Nomination
}

func (db *DB) AddOrUpdateISIN(isin string, source string) error {
	i, err := db.GetISIN(isin)
	if err != nil {
		if !errors.Is(err, storm.ErrNotFound) {
			return err
		}

		i = &ISIN{
			ID:     isin,
			Source: source,
		}
	}

	return db.UpdateFromHTTP(i)
}

func (db *DB) GetAllISIN() ([]ISIN, error) {
	var isin []ISIN

	err := db.DB().All(&isin)

	return isin, err
}

func (db *DB) UpdateShares(isin string) error {
	i, err := db.GetISIN(isin)
	if err != nil {
		return err
	}

	db.logger.Debugf("Calculating shares for ISIN: %s (%.2f)", i.ID, i.Shares)

	ts, err := db.GetTransactionsForISIN(isin)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil
		}

		return err
	}

	newValue := 0.0

	for _, t := range ts {
		newValue += t.TotalShares
	}

	if i.Shares == newValue {
		return nil
	}

	db.logger.Infof("Amount of shares for '%s' changed from %.2f to %.2f", i.ID, i.Shares, newValue)

	i.Shares = newValue

	return db.DB().Save(i)
}
