package main

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func sold(month string, price int) molit.Trade {
	return molit.Trade{AptSeq: "11110-0001", Area: 84.95, Floor: 9, Price: price, Date: month + "-15"}
}

func let(month string, deposit, monthly int, kind string) molit.Rent {
	return molit.Rent{AptSeq: "11110-0001", Area: 84.9, Floor: 9, Deposit: deposit, Monthly: monthly,
		Date: month + "-15", Kind: kind}
}

func ratioHistory() ([]molit.Trade, []molit.Rent) {
	var sales []molit.Trade
	var rents []molit.Rent
	for m := "2023-01"; m <= "2025-12"; m = addMonths(m, 1) {
		price := 100000
		if m >= "2025-01" {
			price = 120000
		}
		sales = append(sales, sold(m, price))
		rents = append(rents,
			let(m, 60000, 0, "신규"),
			let(m, 60000, 0, ""),     // blank kind counts as new
			let(m, 40000, 0, "갱신"),   // renewals are left out
			let(m, 20000, 150, "신규"), // 월세
		)
	}
	return sales, rents
}

func TestJeonseRatio(t *testing.T) {
	sales, rents := ratioHistory()
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	a := analyse(sales, 84, now)
	j := jeonse(a, rents, now)

	if j.RecentJeonse != 60000 || j.RecentLeases == 0 {
		t.Fatalf("recent 전세 %d from %d leases: renewals must not pull it down", j.RecentJeonse, j.RecentLeases)
	}
	if math.Abs(j.Current-50) > 0.01 {
		t.Fatalf("current %.2f, want 50", j.Current)
	}
	// Usual is from months before 2025-10 only: 24 at 60% and 9 at 50%,
	// less the months the ±3 window blends across the change.
	if j.Usual <= 50 || j.Usual >= 60 || !j.Enough {
		t.Fatalf("usual %.2f, enough %v", j.Usual, j.Enough)
	}
	if j.Z >= -1 {
		t.Fatalf("a ratio well under usual should read as dear: z %.2f", j.Z)
	}
	last := j.Points[len(j.Points)-1]
	if !last.After || j.Points[0].After {
		t.Fatalf("after flags: first %+v, last %+v", j.Points[0], last)
	}
}

func TestJeonseYield(t *testing.T) {
	sales, rents := ratioHistory()
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	j := jeonse(analyse(sales, 84, now), rents, now)
	// 150만 a month on 12억 less a 2억 deposit: 1,800 / 100,000 = 1.8%.
	if j.YieldLeases != 12 || math.Abs(j.Yield-1.8) > 0.001 {
		t.Fatalf("yield %.3f from %d leases", j.Yield, j.YieldLeases)
	}
}

func TestJeonseOtherTypesAndLowFloors(t *testing.T) {
	sales, _ := ratioHistory()
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	small := let("2025-12", 30000, 0, "신규")
	small.Area = 59.9
	low := let("2025-12", 30000, 0, "신규")
	low.Floor = 1
	j := jeonse(analyse(sales, 84, now), []molit.Rent{small, low}, now)
	if j.RecentLeases != 0 || len(j.Points) != 0 {
		t.Fatalf("59㎡ and 1층 leases leaked in: %+v", j)
	}
}

func TestJeonseNoRents(t *testing.T) {
	sales, _ := ratioHistory()
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	j := jeonse(analyse(sales, 84, now), nil, now)
	if j.Points == nil || j.Trend == nil || j.Current != 0 || j.Enough {
		t.Fatalf("%+v", j)
	}
	if got := fmt.Sprint(j.Yield, j.YieldLeases); got != "0 0" {
		t.Fatalf("yield %s", got)
	}
}
