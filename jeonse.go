package main

import (
	"math"
	"sort"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

const (
	jeonseRulesFrom = "2025-10"
	yieldSpan       = 12
)

type RatioPoint struct {
	Month  string  `json:"month"`
	Ratio  float64 `json:"ratio"`
	Jeonse int     `json:"jeonse"`
	Sale   int     `json:"sale"`
	After  bool    `json:"after"`
}

type JeonseRatio struct {
	Trend  []TrendPoint `json:"trend"`
	Points []RatioPoint `json:"points"`

	Usual     float64 `json:"usual"`
	Spread    float64 `json:"spread"`
	Current   float64 `json:"current"`
	Z         float64 `json:"z"`
	Enough    bool    `json:"enough"`
	RulesFrom string  `json:"rulesFrom"`

	RecentJeonse int    `json:"recentJeonse"`
	RecentLeases int    `json:"recentLeases"`
	RecentFrom   string `json:"recentFrom"`

	Yield       float64 `json:"yield"`
	YieldLeases int     `json:"yieldLeases"`
}

func leaseReason(r molit.Rent) string {
	switch {
	case r.Kind == "갱신":
		return "renewal"
	case r.Floor <= lowFloor:
		return reasonLowFloor
	}
	return ""
}

func jeonse(a Analysis, rents []molit.Rent, now time.Time) JeonseRatio {
	j := JeonseRatio{Trend: []TrendPoint{}, Points: []RatioPoint{}, RulesFrom: jeonseRulesFrom}
	var full, monthly []molit.Rent
	for _, r := range rents {
		if unitType(r.Area) != a.Area || leaseReason(r) != "" {
			continue
		}
		if r.Monthly == 0 {
			full = append(full, r)
		} else {
			monthly = append(monthly, r)
		}
	}

	// The 전세 line is drawn the way the sales one is, with the deposit as
	// the price.
	rows := make([]Classified, len(full))
	for i, r := range full {
		rows[i] = Classified{Trade: molit.Trade{Date: r.Date, Price: r.Deposit}}
	}
	j.Trend = trend(rows)

	sales := map[string]int{}
	for _, p := range a.Trend {
		sales[p.Month] = p.Median
	}
	var before []float64
	for _, p := range j.Trend {
		s := sales[p.Month]
		if p.Median == 0 || s == 0 {
			continue
		}
		rp := RatioPoint{Month: p.Month, Ratio: float64(p.Median) / float64(s) * 100,
			Jeonse: p.Median, Sale: s, After: p.Month >= jeonseRulesFrom}
		j.Points = append(j.Points, rp)
		if !rp.After {
			before = append(before, rp.Ratio)
		}
	}
	j.Usual, j.Spread = meanSD(before)
	j.Enough = len(before) >= relativeMinMonths

	recent := summarise(rows, nil, now)
	j.RecentJeonse, j.RecentLeases, j.RecentFrom = recent.RecentMedian, recent.RecentSales, recent.RecentFrom
	if j.RecentJeonse > 0 && a.Summary.RecentMedian > 0 {
		j.Current = float64(j.RecentJeonse) / float64(a.Summary.RecentMedian) * 100
		if j.Spread > 0 {
			j.Z = (j.Current - j.Usual) / j.Spread
		}
	}

	from := now.AddDate(0, -yieldSpan, 0).Format(time.DateOnly)
	var yields []float64
	for _, r := range monthly {
		if r.Date < from {
			continue
		}
		price := saleAround(sales, r.Date[:7], a.Summary.RecentMedian)
		if in := price - r.Deposit; in > 0 {
			yields = append(yields, float64(r.Monthly*12)/float64(in)*100)
		}
	}
	j.YieldLeases = len(yields)
	j.Yield = medianFloat(yields)
	return j
}

func saleAround(sales map[string]int, month string, fallback int) int {
	for d := 0; d <= 3; d++ {
		if p := sales[addMonths(month, -d)]; p > 0 {
			return p
		}
	}
	return fallback
}

func meanSD(xs []float64) (mean, sd float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	for _, x := range xs {
		sd += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(sd / float64(len(xs)))
}

func medianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
