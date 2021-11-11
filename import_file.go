package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func walkAllFiles(db *DB) {
	filepath.Walk("output/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			db.logger.Fatalf(err.Error())
		}

		if !strings.HasSuffix(path, ".tbl") {
			return nil
		}

		isin := strings.SplitN(info.Name(), ".", 2)[0]
		db.logger.Infof("Now parsing: %s (ISIN %s)\n", path, isin)

		res, err := readFile(isin, path)
		if err != nil {
			db.logger.Warn(err)
			return nil
		}

		i, err := db.GetISIN(isin)
		if err != nil {
			return err
		}

		return db.ImportValuations(i, res)
	})
}

func readFile(isin string, filename string) ([]*Valuation, error) {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var in struct {
		HTML string `json:"html"`
	}

	if err := json.Unmarshal(input, &in); err != nil {
		return nil, err
	}

	res, err := parseFTInput(isin, in.HTML)
	if err != nil {
		return nil, err
	}

	return res, nil
}
