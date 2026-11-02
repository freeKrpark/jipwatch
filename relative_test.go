package main

import (
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

// sales makes one normal sale a month from start for each price.
func sales(start string, prices ...int) []molit.Trade {
	var out []molit.Trade
	for i, p := range prices {
		out = append(out, molit.Trade{Date: addMonths(start, i) + "-15", Price: p, Area: 84.95, Floor: 9})
	}
	return out
}

func repeat(n, v int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func TestRelativeUsualPremiumThenDearer(t *testing.T) {
	// Two years at 110 against a district at 100: a usual premium of 10%.
	// The last eight months the complex trades at 125 while the district
	// stays at 100, a premium well above usual.
	own := sales("2024-01", append(repeat(24, 110), repeat(8, 125)...)...)
	district := sales("2024-01", repeat(32, 100)...)
	now := mustTime("2026-08-20")
	a := analyse(own, 0, now)
	r := relative(a, district, now)

	if !r.Enough || r.BandMin != 79 || r.BandMax != 88 {
		t.Fatalf("%+v", r)
	}
	if r.Current < 24 || r.Current > 26 {
		t.Fatalf("current premium %.1f, want 25", r.Current)
	}
	if r.Usual < 10 || r.Usual > 16 {
		t.Fatalf("usual premium %.1f", r.Usual)
	}
	if r.Z < 1 {
		t.Fatalf("z %.2f: trading well above its usual premium should show", r.Z)
	}
}

func TestRelativeNeedsHistory(t *testing.T) {
	own := sales("2026-03", repeat(6, 110)...)
	district := sales("2026-03", repeat(6, 100)...)
	now := mustTime("2026-08-20")
	if r := relative(analyse(own, 0, now), district, now); r.Enough {
		t.Fatalf("six months is not enough to know what is usual: %+v", r)
	}
	if r := relative(analyse(own, 0, now), nil, now); r.Enough || len(r.Points) != 0 || r.Points == nil {
		t.Fatalf("no district sales: %+v", r)
	}
}

func mustTime(day string) time.Time {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		panic(err)
	}
	return t
}
