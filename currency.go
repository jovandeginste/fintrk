package main

import (
	"github.com/cloudfoundry-attic/jibber_jabber"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

type Currency struct {
	printer *message.Printer
}

func (c *Currency) Initialize() error {
	l, err := jibber_jabber.DetectLanguage()
	if err != nil {
		return err
	}

	lang, err := language.Parse(l)
	if err != nil {
		return err
	}

	c.printer = message.NewPrinter(lang)

	return nil
}

func (c *Currency) Localize(code string, valuation float64) (string, error) {
	cur, err := currency.ParseISO(code)
	if err != nil {
		return "", err
	}

	scale, _ := currency.Cash.Rounding(cur) // fractional digits
	dec := number.Decimal(valuation, number.Scale(scale))

	return c.printer.Sprintf("%v %v", currency.Symbol(cur), dec), nil
}
