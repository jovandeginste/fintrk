package main

import (
	"fmt"
	"time"

	"github.com/asdine/storm/v3/q"
)

type Valuation struct {
	ID    string `storm:"id"`
	ISIN  string `storm:"index"`
	Date  time.Time
	Open  float64
	High  float64
	Low   float64
	Close float64
}

func (v *Valuation) Value() float64 {
	return v.Open
}

func (db *DB) GetValuation(id string) (*Valuation, error) {
	var i Valuation

	db.logger.Debugf("Searching for ISIN valuation for '%s'", id)

	query := db.DB().Select(
		q.Eq("ISIN", id),
	).Reverse().OrderBy("Date").Limit(1)

	if err := query.First(&i); err != nil {
		return nil, err
	}

	db.logger.Debugf("Valuation for '%s': %#v", id, i)

	return &i, nil
}

func (db *DB) GetValuationAt(isin string, d time.Time) (*Valuation, error) {
	var v Valuation

	query := db.DB().Select(
		q.Eq("ISIN", isin),
		q.Lte("Date", d),
	).Reverse().OrderBy("Date").Limit(1)

	if err := query.First(&v); err != nil {
		return nil, err
	}

	return &v, nil
}

func (db *DB) GetSharesAt(isin string, d time.Time) (float64, error) {
	var transactions []Transaction

	query := db.DB().Select(
		q.Eq("ISIN", isin),
		q.Lte("Date", d),
	).Reverse().OrderBy("Date")

	if err := query.Find(&transactions); err != nil {
		return 0, err
	}

	var v float64

	for _, tx := range transactions {
		v += tx.TotalShares
	}

	return v, nil
}

func (v *Valuation) UpdateID() {
	v.ID = fmt.Sprintf("%s@%s", v.ISIN, timeToDate(&v.Date))
}
