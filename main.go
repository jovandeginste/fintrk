package main

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

type App struct {
	db       *DB
	logger   *logrus.Logger
	currency *Currency
}

func (a *App) DB() *DB {
	return a.db
}

func (a *App) Logger() *logrus.Logger {
	return a.logger
}

func main() {
	DBFile := ""

	if os.Getenv("FINTRK_DBFILE") != "" {
		DBFile = os.Getenv("FINTRK_DBFILE")
	}

	if DBFile == "" {
		d, err := homedir.Dir()
		if err != nil {
			panic(err)
		}

		DBFile = filepath.Join(d, ".fintrk.db")
	}

	l := logrus.StandardLogger()
	a := App{
		db:       NewDB(DBFile, l),
		logger:   l,
		currency: &Currency{},
	}

	if err := a.currency.Initialize(); err != nil {
		a.logger.Fatal(err)
	}

	if err := a.DB().Initialize(); err != nil {
		a.logger.Fatalf("%v", err)
	}

	defer a.DB().Close()

	cmd := a.RootCmd()
	if err := cmd.Execute(); err != nil {
		a.logger.Fatal(err)
	}
}
