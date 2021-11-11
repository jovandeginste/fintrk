package main

import (
	"github.com/asdine/storm/v3"
	"github.com/remeh/sizedwaitgroup"
	"github.com/sirupsen/logrus"
)

type DB struct {
	db *storm.DB

	file   string
	logger *logrus.Logger
}

func NewDB(file string, logger *logrus.Logger) *DB {
	db := DB{
		file:   file,
		logger: logger,
	}

	return &db
}

func (db *DB) DB() *storm.DB {
	return db.db
}

func (db *DB) Initialize() error {
	myDB, err := storm.Open(db.file)
	if err != nil {
		return err
	}

	dbBacked := []interface{}{Valuation{}, ISIN{}, Transaction{}}

	for _, e := range dbBacked {
		if err := myDB.Init(e); err != nil {
			return err
		}
	}

	db.db = myDB

	return nil
}

func (db *DB) Close() {
	db.DB().Close()
}

func (db *DB) UpdateValuationsAll() error {
	isin, err := db.GetAllISIN()
	if err != nil {
		return err
	}

	swg := sizedwaitgroup.New(8)

	for n := range isin {
		swg.Add()

		go func(i *ISIN) {
			defer swg.Done()
			db.logger.Infof("Updating ISIN: %s", i.ID)

			if err := db.UpdateFromHTTP(i); err != nil {
				db.logger.Errorf("Error updating ISIN '%s': %v", i.ID, err)
			}
		}(&isin[n])
	}

	swg.Wait()

	return nil
}
