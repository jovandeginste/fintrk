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

	query := db.DB().Select(
		q.Eq("ID", id),
	).Limit(1)

	if err := query.First(&i); err != nil {
		return nil, err
	}

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

func (v *Valuation) UpdateID() {
	v.ID = fmt.Sprintf("%s@%s", v.ISIN, timeToDate(&v.Date))
}
