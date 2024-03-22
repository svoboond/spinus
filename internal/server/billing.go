package server

import "time"

type Reading struct {
	Value float64
	Time  time.Time
}
