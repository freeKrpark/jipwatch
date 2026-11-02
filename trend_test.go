package main

import (
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func TestMedian(t *testing.T) {
	cases := []struct {
		in   []int
		want int
	}{{nil, 0}, {[]int{5}, 5}, {[]int{9, 1, 5}, 5}, {[]int{4, 1, 3, 2}, 2}}
	for _, c := range cases {
		if got := median(c.in); got != c.want {
			t.Errorf("median(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// monthly makes one normal sale per month from start, at the given prices.
func monthly(start string, prices ...int) []Classified {
	var rows []Classified
	for i, p := range prices {
		m := addMonths(start, i)
		rows = append(rows, Classified{Trade: molit.Trade{Date: m + "-15", Price: p}})
	}
	return rows
}

func TestTrendFindsPeakThenTrough(t *testing.T) {
	// Up to a peak of 130 in 2021-10, down to 100 in 2023-02, back to 120.
	var prices []int
	for i := 0; i < 10; i++ { // 2021-01..2021-10
		prices = append(prices, 110+i*2)
	}
	for i := 1; i <= 16; i++ { // 2021-11..2023-02
		prices = append(prices, 128-i*2)
	}
	for i := 1; i <= 10; i++ { // 2023-03..2023-12
		prices = append(prices, 96+i*2)
	}
	rows := monthly("2021-01", prices...)
	// An odd low sale and an excluded one must not move the trough.
	rows = append(rows,
		Classified{Trade: molit.Trade{Date: "2023-01-20", Price: 60}},
		Classified{Trade: molit.Trade{Date: "2023-02-01", Price: 50}, Reason: reasonDirect})

	points := trend(rows)
	if len(points) != 36 || points[0].Month != "2021-01" || points[35].Month != "2023-12" {
		t.Fatalf("%d points, %s..%s", len(points), points[0].Month, points[len(points)-1].Month)
	}
	s := summarise(rows, points, time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC))
	if s.Peak == nil || s.Peak.Month < "2021-08" || s.Peak.Month > "2021-11" {
		t.Fatalf("peak %+v", s.Peak)
	}
	if s.Trough == nil || s.Trough.Month < "2022-12" || s.Trough.Month > "2023-03" || s.Trough.Price < 96 {
		t.Fatalf("trough %+v", s.Trough)
	}
	// Recent: 2023-07-10 onwards, the last six sales 106..116.
	if s.RecentSales != 6 || s.RecentMedian != 111 {
		t.Fatalf("recent %d sales, median %d", s.RecentSales, s.RecentMedian)
	}
	if s.FromTrough <= 0 || s.FromPeak >= 0 {
		t.Fatalf("from peak %.1f%%, from trough %.1f%%", s.FromPeak, s.FromTrough)
	}
}

// Thin stretches break the line instead of drawing through them.
func TestTrendBreaksWhereSalesAreThin(t *testing.T) {
	rows := monthly("2020-01", 100, 100, 100, 100)
	rows = append(rows, Classified{Trade: molit.Trade{Date: "2022-06-15", Price: 150}})
	points := trend(rows)
	for _, p := range points {
		if p.Month == "2021-06" && p.Median != 0 {
			t.Fatalf("2021-06 has no sales nearby but got %+v", p)
		}
		if p.Month == "2020-02" && p.Median != 100 {
			t.Fatalf("2020-02: %+v", p)
		}
	}
	s := summarise(rows, points, time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC))
	if s.Peak == nil || s.Peak.Price != 100 {
		t.Fatalf("a lone sale must not be the peak: %+v", s.Peak)
	}
}

func TestSummariseWithoutSales(t *testing.T) {
	s := summarise(nil, trend(nil), time.Now())
	if s.Peak != nil || s.Trough != nil || s.RecentMedian != 0 || s.FromPeak != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestAddMonths(t *testing.T) {
	if got := addMonths("2023-11", 3); got != "2024-02" {
		t.Errorf("got %s", got)
	}
	if got := addMonths("2023-02", -3); got != "2022-11" {
		t.Errorf("got %s", got)
	}
}

// In a slump the window widens until it holds enough sales, and says so.
func TestTrendWidensWhenSalesAreThin(t *testing.T) {
	// A sale every month, then one every four months, then monthly again.
	rows := monthly("2022-01", 100, 100, 100, 100, 100, 100)
	for _, m := range []string{"2022-09", "2023-01", "2023-05", "2023-09"} {
		rows = append(rows, Classified{Trade: molit.Trade{Date: m + "-15", Price: 90}})
	}
	rows = append(rows, monthly("2023-12", 95, 95, 95, 95, 95, 95)...)

	byMonth := map[string]TrendPoint{}
	for _, p := range trend(rows) {
		byMonth[p.Month] = p
	}
	if p := byMonth["2022-03"]; p.Radius != trendRadius || p.Wide() {
		t.Fatalf("busy months keep the usual window: %+v", p)
	}
	slump := byMonth["2023-03"]
	if slump.Median != 90 || !slump.Wide() || slump.Sales < trendMinSales {
		t.Fatalf("2023-03 has two sales within three months; it should widen: %+v", slump)
	}
	s := summarise(rows, trend(rows), mustTime("2024-06-20"))
	if s.Trough == nil || s.Trough.Price != 90 || !s.Trough.Wide {
		t.Fatalf("trough from a widened window should say so: %+v", s.Trough)
	}
}

// Once prices pass the old peak, the downturn is still the 2021-23 fall,
// not a dip after the new high.
func TestTroughIsTheDeepestFallNotAfterTheNewHigh(t *testing.T) {
	var prices []int
	prices = append(prices, repeat(6, 120)...) // old peak
	prices = append(prices, repeat(6, 90)...)  // the downturn
	prices = append(prices, repeat(6, 130)...) // a new high
	prices = append(prices, repeat(6, 127)...) // a small dip after it
	rows := monthly("2021-01", prices...)
	s := summarise(rows, trend(rows), mustTime("2023-01-01"))
	if s.Trough == nil || s.Trough.Price != 90 {
		t.Fatalf("trough %+v, want the 90 downturn", s.Trough)
	}
	if s.Peak == nil || s.Peak.Price != 120 {
		t.Fatalf("peak %+v, want the 120 before the downturn", s.Peak)
	}
}

func TestNoTroughWhenPricesNeverFell(t *testing.T) {
	rows := monthly("2021-01", 100, 101, 102, 103, 104, 105, 106, 107, 108)
	s := summarise(rows, trend(rows), mustTime("2021-10-01"))
	if s.Trough != nil || s.Peak == nil {
		t.Fatalf("%+v", s)
	}
}
