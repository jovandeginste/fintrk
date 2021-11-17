package main

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

type App struct {
	Debug bool

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
	app := App{
		db:       NewDB(DBFile, l),
		logger:   l,
		currency: &Currency{},
	}

	if err := app.currency.Initialize(); err != nil {
		app.logger.Fatal(err)
	}

	if err := app.DB().Initialize(); err != nil {
		app.logger.Fatalf("%v", err)
	}

	defer app.DB().Close()

	cmd := app.RootCmd()
	if err := cmd.Execute(); err != nil {
		app.logger.Fatal(err)
	}
}
