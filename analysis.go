package main

import (
	"math"
	"sort"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

const lowFloor = 2

const (
	reasonCanceled = "canceled"  // the contract was undone
	reasonDirect   = "direct"    // no broker: often family, at a non-market price
	reasonLowFloor = "low_floor" // 1~2층
)

func unitType(area float64) int { return int(math.Floor(area)) }

func reject(t molit.Trade) string {
	switch {
	case t.Canceled:
		return reasonCanceled
	case t.Direct:
		return reasonDirect
	case t.Floor <= lowFloor:
		return reasonLowFloor
	}
	return ""
}

type TypeCount struct {
	Area  int `json:"area"`
	Sales int `json:"sales"`
}

type Classified struct {
	Trade  molit.Trade `json:"trade"`
	Reason string      `json:"reason"`
}

type Analysis struct {
	Types  []TypeCount  `json:"types"`
	Area   int          `json:"area"`
	Trades []Classified `json:"trades"`

	Trend   []TrendPoint `json:"trend"`
	Summary Summary      `json:"summary"`
}

func analyse(history []molit.Trade, area int, now time.Time) Analysis {
	counts := map[int]int{}
	for _, t := range history {
		n := counts[unitType(t.Area)]
		if reject(t) == "" {
			n++
		}
		counts[unitType(t.Area)] = n
	}
	a := Analysis{Types: []TypeCount{}, Trades: []Classified{}}
	for ar, n := range counts {
		a.Types = append(a.Types, TypeCount{Area: ar, Sales: n})
	}
	sort.Slice(a.Types, func(i, j int) bool {
		if a.Types[i].Sales != a.Types[j].Sales {
			return a.Types[i].Sales > a.Types[j].Sales
		}
		return a.Types[i].Area < a.Types[j].Area
	})
	a.Area = area
	if _, ok := counts[area]; !ok && len(a.Types) > 0 {
		a.Area = a.Types[0].Area
	}
	for _, t := range history {
		if unitType(t.Area) == a.Area {
			a.Trades = append(a.Trades, Classified{Trade: t, Reason: reject(t)})
		}
	}
	a.Trend = trend(a.Trades)
	a.Summary = summarise(a.Trades, a.Trend, now)
	return a
}
