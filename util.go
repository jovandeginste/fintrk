package main

import (
	"fmt"
	"time"
)

func timeToDate(t *time.Time) string {
	y, m, d := t.UTC().Date()
	return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
}
