package main

import (
	"github.com/freekrpark/jipwatch/internal/store"
)

const m2MinMonths = 12

type M2Point struct {
	Month    string  `json:"month"`
	Nominal  int     `json:"nominal"`
	M2       float64 `json:"m2"`
	Adjusted float64 `json:"adjusted"`
	Rate     float64 `json:"rate"`
}

type M2Analysis struct {
	Points  []M2Point `json:"points"`
	Usual   float64   `json:"usual"`
	Spread  float64   `json:"spread"`
	Current float64   `json:"current"`
	Z       float64   `json:"z"`
	Enough  bool      `json:"enough"`
}

func m2analysis(trendPts []TrendPoint, m2data, rateData []store.EcosPoint) M2Analysis {
	a := M2Analysis{Points: []M2Point{}}

	m2map := make(map[string]float64, len(m2data))
	for _, p := range m2data {
		if len(p.YM) == 6 {
			m2map[p.YM[:4]+"-"+p.YM[4:]] = p.Value
		}
	}
	ratemap := make(map[string]float64, len(rateData))
	for _, p := range rateData {
		if len(p.YM) == 6 {
			ratemap[p.YM[:4]+"-"+p.YM[4:]] = p.Value
		}
	}

	if len(m2map) == 0 {
		return a
	}

	var baseM2 float64
	for _, p := range trendPts {
		if p.Median == 0 {
			continue
		}
		if m, ok := m2map[p.Month]; ok && m > 0 {
			baseM2 = m
			break
		}
	}
	if baseM2 == 0 {
		return a
	}

	for _, p := range trendPts {
		if p.Median == 0 {
			continue
		}
		m, ok := m2map[p.Month]
		if !ok || m == 0 {
			continue
		}
		a.Points = append(a.Points, M2Point{
			Month:    p.Month,
			Nominal:  p.Median,
			M2:       m,
			Adjusted: float64(p.Median) / m * baseM2,
			Rate:     ratemap[p.Month],
		})
	}

	if len(a.Points) == 0 {
		return a
	}

	vals := make([]float64, len(a.Points))
	for i, p := range a.Points {
		vals[i] = p.Adjusted
	}
	a.Usual, a.Spread = meanSD(vals)
	a.Current = a.Points[len(a.Points)-1].Adjusted
	if a.Spread > 0 {
		a.Z = (a.Current - a.Usual) / a.Spread
	}
	a.Enough = len(a.Points) >= m2MinMonths
	return a
}
