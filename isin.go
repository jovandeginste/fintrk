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

func (db *DB) AddOrUpdateISIN(isinID string, source string) error {
	isin, err := db.GetISIN(isinID)
	if err != nil {
		if !errors.Is(err, storm.ErrNotFound) {
			return err
		}

		isin = &ISIN{
			ID:     isinID,
			Source: source,
		}
	}

	return db.UpdateFromHTTP(isin)
}

func (db *DB) GetAllISIN() ([]ISIN, error) {
	var isin []ISIN

	err := db.DB().All(&isin)

	return isin, err
}

func (db *DB) UpdateShares(isinID string) error {
	isin, err := db.GetISIN(isinID)
	if err != nil {
		return err
	}

	db.logger.Debugf("Calculating shares for ISIN: %s (%.2f)", isin.ID, isin.Shares)

	transactions, err := db.GetTransactionsForISIN(isinID)
	if err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil
		}

		return err
	}

	newValue := 0.0

	for _, t := range transactions {
		newValue += t.TotalShares
	}

	if isin.Shares == newValue {
		return nil
	}

	db.logger.Infof("Amount of shares for '%s' changed from %.2f to %.2f", isin.ID, isin.Shares, newValue)

	isin.Shares = newValue

	return db.DB().Save(isin)
}
