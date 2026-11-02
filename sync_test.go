package main

import (
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

// months counts the months from historyFrom through now's month.
func months(now time.Time) int {
	start, _ := time.Parse("200601", historyFrom)
	return (now.Year()-start.Year())*12 + int(now.Month()-start.Month()) + 1
}

func TestSyncPlanFirstTime(t *testing.T) {
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	plan := syncPlan(map[string]string{}, now)
	if len(plan) != months(now) || plan[0] != "202611" || plan[len(plan)-1] != historyFrom {
		t.Fatalf("%d months, %s..%s", len(plan), plan[0], plan[len(plan)-1])
	}
}

func TestSyncPlanRefetchesOnlyRecentAndMissing(t *testing.T) {
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	fetched := map[string]string{}
	for _, ym := range syncPlan(fetched, now) {
		fetched[ym] = "earlier"
	}
	delete(fetched, "202304") // a month that failed last time

	got := syncPlan(fetched, now)
	want := []string{"202611", "202610", "202609", "202304"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestCoverage(t *testing.T) {
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	c := coverage(map[string]string{"202611": "x", "202304": "x", "199912": "before historyFrom"}, now)
	if c.Fetched != 2 || c.Total != months(now) || c.From != historyFrom {
		t.Fatalf("%+v", c)
	}
}

func TestWatchItemSkipsCanceled(t *testing.T) {
	history := []molit.Trade{
		{Date: "2023-04-01", Price: 90000},
		{Date: "2023-05-01", Price: 95000},
		{Date: "2023-06-01", Price: 70000, Canceled: true},
	}
	w := watchItem("a", "b", "c", "d", history)
	if w.Sales != 2 || w.Latest == nil || w.Latest.Price != 95000 {
		t.Fatalf("%+v", w)
	}
	if w := watchItem("a", "b", "c", "d", nil); w.Latest != nil || w.Sales != 0 {
		t.Fatalf("empty: %+v", w)
	}
}
