package server

import (
	"time"
)

type BreakPoints [][3]time.Time

func (bp BreakPoints) Less(i, j int) bool { return bp[i][1].Before(bp[j][1]) }
func (bp BreakPoints) Swap(i, j int)      { bp[i], bp[j] = bp[j], bp[i] }
func (bp BreakPoints) Len() int           { return len(bp) }

type Reading struct {
	Value float64
	Time  time.Time
	Valid bool
}
