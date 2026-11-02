package main

import (
	"sort"
	"time"
)

const (
	trendRadius   = 3
	trendMinSales = 3
	recentSpan    = 6
)

type TrendPoint struct {
	Month  string `json:"month"`
	Median int    `json:"median"`
	Sales  int    `json:"sales"`
}

type Mark struct {
	Month string `json:"month"`
	Price int    `json:"price"`
}

type Summary struct {
	Peak         *Mark   `json:"peak"`
	Trough       *Mark   `json:"trough"`
	RecentMedian int     `json:"recentMedian"`
	RecentSales  int     `json:"recentSales"`
	RecentFrom   string  `json:"recentFrom"`
	FromPeak     float64 `json:"fromPeak"`
	FromTrough   float64 `json:"fromTrough"`
}

func median(prices []int) int {
	if len(prices) == 0 {
		return 0
	}
	s := append([]int(nil), prices...)
	sort.Ints(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func addMonths(month string, n int) string {
	t, _ := time.Parse("2006-01", month)
	return t.AddDate(0, n, 0).Format("2006-01")
}

func trend(rows []Classified) []TrendPoint {
	byMonth := map[string][]int{}
	first, last := "", ""
	for _, r := range rows {
		if r.Reason != "" {
			continue
		}
		m := r.Trade.Date[:7]
		byMonth[m] = append(byMonth[m], r.Trade.Price)
		if first == "" || m < first {
			first = m
		}
		if m > last {
			last = m
		}
	}
	points := []TrendPoint{}
	if first == "" {
		return points
	}
	for m := first; m <= last; m = addMonths(m, 1) {
		var window []int
		for d := -trendRadius; d <= trendRadius; d++ {
			window = append(window, byMonth[addMonths(m, d)]...)
		}
		p := TrendPoint{Month: m, Sales: len(window)}
		if len(window) >= trendMinSales {
			p.Median = median(window)
		}
		points = append(points, p)
	}
	return points
}

func summarise(rows []Classified, points []TrendPoint, now time.Time) Summary {
	var s Summary
	peakAt := -1
	for i, p := range points {
		if p.Median > 0 && (peakAt < 0 || p.Median > points[peakAt].Median) {
			peakAt = i
		}
	}
	if peakAt >= 0 {
		s.Peak = &Mark{points[peakAt].Month, points[peakAt].Median}
		for _, p := range points[peakAt+1:] {
			if p.Median > 0 && (s.Trough == nil || p.Median < s.Trough.Price) {
				s.Trough = &Mark{p.Month, p.Median}
			}
		}
	}

	s.RecentFrom = now.AddDate(0, -recentSpan, 0).Format("2006-01-02")
	var recent []int
	for _, r := range rows {
		if r.Reason == "" && r.Trade.Date >= s.RecentFrom {
			recent = append(recent, r.Trade.Price)
		}
	}
	s.RecentSales = len(recent)
	s.RecentMedian = median(recent)
	if s.RecentMedian > 0 && s.Peak != nil {
		s.FromPeak = pct(s.RecentMedian, s.Peak.Price)
	}
	if s.RecentMedian > 0 && s.Trough != nil {
		s.FromTrough = pct(s.RecentMedian, s.Trough.Price)
	}
	return s
}

func pct(now, then int) float64 {
	return float64(now-then) / float64(then) * 100
}
