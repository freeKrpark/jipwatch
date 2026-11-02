package main

import (
	"math"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

const (
	bandHalf          = 5
	relativeMinMonths = 12
)

type PremiumPoint struct {
	Month   string  `json:"month"`
	Premium float64 `json:"premium"`
}

type Relative struct {
	BandMin int            `json:"bandMin"`
	BandMax int            `json:"bandMax"`
	Points  []PremiumPoint `json:"points"`
	Usual   float64        `json:"usual"`
	Spread  float64        `json:"spread"`
	Current float64        `json:"current"`
	Z       float64        `json:"z"`
	Enough  bool           `json:"enough"`
}

func relative(a Analysis, district []molit.Trade, now time.Time) Relative {
	r := Relative{BandMin: a.Area - bandHalf, BandMax: a.Area + bandHalf - 1, Points: []PremiumPoint{}}
	var rows []Classified
	for _, t := range district {
		rows = append(rows, Classified{Trade: t, Reason: reject(t)})
	}
	districtTrend := map[string]int{}
	for _, p := range trend(rows) {
		districtTrend[p.Month] = p.Median
	}
	var sum float64
	for _, p := range a.Trend {
		d := districtTrend[p.Month]
		if p.Median == 0 || d == 0 {
			continue
		}
		prem := pct(p.Median, d)
		r.Points = append(r.Points, PremiumPoint{Month: p.Month, Premium: prem})
		sum += prem
	}
	if len(r.Points) == 0 {
		return r
	}
	r.Usual = sum / float64(len(r.Points))
	var sq float64
	for _, p := range r.Points {
		sq += (p.Premium - r.Usual) * (p.Premium - r.Usual)
	}
	r.Spread = math.Sqrt(sq / float64(len(r.Points)))

	districtRecent := summarise(rows, nil, now).RecentMedian
	if a.Summary.RecentMedian > 0 && districtRecent > 0 {
		r.Current = pct(a.Summary.RecentMedian, districtRecent)
		if r.Spread > 0 {
			r.Z = (r.Current - r.Usual) / r.Spread
		}
		r.Enough = len(r.Points) >= relativeMinMonths
	}
	return r
}
