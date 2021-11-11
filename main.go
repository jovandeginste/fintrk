package main

import (
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
	l := logrus.StandardLogger()
	a := App{
		db:       NewDB("fintrk.db", l),
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
