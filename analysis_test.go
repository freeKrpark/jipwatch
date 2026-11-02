package main

import (
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func tr(date string, area float64, floor, price int, mods ...func(*molit.Trade)) molit.Trade {
	t := molit.Trade{Date: date, Area: area, Floor: floor, Price: price}
	for _, m := range mods {
		m(&t)
	}
	return t
}

var testNow = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

func direct(t *molit.Trade)       { t.Direct = true }
func canceledSale(t *molit.Trade) { t.Canceled = true }

func TestReject(t *testing.T) {
	cases := map[string]molit.Trade{
		"":             tr("2023-04-01", 84.95, 3, 99000),
		reasonLowFloor: tr("2023-04-01", 84.95, 2, 90000),
		reasonDirect:   tr("2023-04-01", 84.95, 9, 65000, direct),
		reasonCanceled: tr("2023-04-01", 84.95, 9, 99000, canceledSale, direct),
	}
	for want, trade := range cases {
		if got := reject(trade); got != want {
			t.Errorf("%+v: got %q, want %q", trade, got, want)
		}
	}
}

func TestAnalysePicksBusiestTypeAndKeepsExcluded(t *testing.T) {
	history := []molit.Trade{
		tr("2021-09-10", 84.95, 15, 128000),
		tr("2022-01-05", 59.98, 7, 90000),
		tr("2023-03-02", 84.81, 8, 65000, direct),
		tr("2023-04-20", 84.99, 9, 99000),
		tr("2023-05-01", 114.98, 6, 120000),
	}
	a := analyse(history, 0, testNow)
	if a.Area != 84 {
		t.Fatalf("area %d, want 84 (two normal sales)", a.Area)
	}
	if len(a.Types) != 3 || a.Types[0] != (TypeCount{84, 2}) {
		t.Fatalf("types %+v", a.Types)
	}
	if len(a.Trades) != 3 || a.Trades[1].Reason != reasonDirect || a.Trades[0].Reason != "" {
		t.Fatalf("trades %+v", a.Trades)
	}

	if a := analyse(history, 59, testNow); a.Area != 59 || len(a.Trades) != 1 {
		t.Fatalf("asked for 59: %+v", a)
	}
	if a := analyse(history, 77, testNow); a.Area != 84 {
		t.Fatalf("unknown type should fall back to the busiest, got %d", a.Area)
	}
}

func TestAnalyseEmpty(t *testing.T) {
	a := analyse(nil, 0, testNow)
	if a.Types == nil || a.Trades == nil || a.Area != 0 {
		t.Fatalf("%+v", a)
	}
}
