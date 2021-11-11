package main

import (
	"fmt"
	"time"

	"github.com/asdine/storm/v3/q"
	"github.com/google/uuid"
)

type Transaction struct {
	UUID        uuid.UUID `storm:"id"`
	Date        time.Time `storm:"index"`
	ISIN        string    `storm:"index"`
	TotalShares float64
	TotalValue  float64
}

func (t *Transaction) ValuePerShare() float64 {
	return t.TotalValue / t.TotalShares
}

func (t *Transaction) GenerateUUID() {
	if t.UUID != uuid.Nil {
		return
	}

	t.UUID = uuid.New()
}

func (db *DB) GetTransaction(d time.Time) (*Transaction, error) {
	var i Transaction

	query := db.DB().Select(
		q.Eq("Date", d),
	).Limit(1)

	if err := query.First(&i); err != nil {
		return nil, err
	}

	return &i, nil
}

func (db *DB) GetTransactionsForISIN(isin string) ([]Transaction, error) {
	var i []Transaction

	query := db.DB().Select(
		q.Eq("ISIN", isin),
	)

	if err := query.Find(&i); err != nil {
		return nil, err
	}

	return i, nil
}

func (db *DB) CreateTransaction(t *Transaction) error {
	t.GenerateUUID()

	return db.DB().Save(t)
}

func (t *Transaction) String() string {
	return fmt.Sprintf(
		"ISIN: '%v'; total value: %.2f, shares: '%.2f', date: %s",
		t.ISIN,
		t.TotalValue,
		t.TotalShares,
		timeToDate(&t.Date),
	)
}

func (t *Transaction) SetDate(d string) error {
	if d == "" {
		t.Date = time.Now()
		return nil
	}

	parsed, err := time.Parse("2006-01-02", d)
	if err != nil {
		return err
	}

	t.Date = parsed

	return nil
}
