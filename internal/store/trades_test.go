package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func sale(day string, price int, mods ...func(*molit.Trade)) molit.Trade {
	t := molit.Trade{AptSeq: "11110-0001", AptNm: "첫째아파트", SggCd: "11110", UmdNm: "테스트동",
		Area: 84.95, Floor: 9, Price: price, Date: "2023-04-" + day, RegisteredOn: "2023-06-01", BuildYear: 2011}
	for _, m := range mods {
		m(&t)
	}
	return t
}

func canceled(t *molit.Trade) { t.Canceled, t.CanceledOn, t.RegisteredOn = true, "2023-04-27", "" }

func TestMatch(t *testing.T) {
	old := []saved{
		{Trade: sale("09", 90000), firstSeen: "T1"},
		{Trade: sale("12", 95000), firstSeen: "T1"},
	}
	fresh := []molit.Trade{
		sale("09", 90000),           // unchanged
		sale("09", 90000, canceled), // the 09 sale's cancel record arrived: re-reported pair
		sale("12", 95000, canceled), // the 12 sale was canceled outright
		sale("20", 99000),           // new sale
	}
	rows, ch := match(old, fresh, "T2")

	if len(ch.Added) != 1 || ch.Added[0].Date != "2023-04-20" {
		t.Fatalf("added: %+v", ch.Added)
	}
	if len(ch.Canceled) != 2 {
		t.Fatalf("canceled: %+v", ch.Canceled)
	}
	want := []string{"T1", "T2", "T2", "T2"}
	for i, r := range rows {
		if r.firstSeen != want[i] {
			t.Errorf("row %d first seen %s, want %s", i, r.firstSeen, want[i])
		}
	}
}

// Two identical live rows stay two sales, not one.
func TestMatchCountsDuplicates(t *testing.T) {
	old := []saved{{Trade: sale("09", 90000), firstSeen: "T1"}}
	_, ch := match(old, []molit.Trade{sale("09", 90000), sale("09", 90000)}, "T2")
	if len(ch.Added) != 1 {
		t.Fatalf("added: %+v", ch.Added)
	}
}

func TestReplaceMonth(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	t1 := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)

	if trades, at, err := s.Month(ctx, "11110", "202304"); err != nil || at != "" || trades != nil {
		t.Fatalf("never fetched: %v %q %v", trades, at, err)
	}

	ch, err := s.ReplaceMonth(ctx, "11110", "202304", []molit.Trade{sale("12", 95000), sale("09", 90000)}, t1)
	if err != nil || !ch.First || len(ch.Added) != 0 {
		t.Fatalf("first fetch: %+v %v", ch, err)
	}

	ch, err = s.ReplaceMonth(ctx, "11110", "202304",
		[]molit.Trade{sale("09", 90000), sale("09", 90000, canceled), sale("12", 95000), sale("20", 99000)}, t2)
	if err != nil || ch.First || len(ch.Added) != 1 || len(ch.Canceled) != 1 {
		t.Fatalf("second fetch: %+v %v", ch, err)
	}

	trades, at, err := s.Month(ctx, "11110", "202304")
	if err != nil {
		t.Fatal(err)
	}
	if at != "2026-11-02 09:00" || len(trades) != 4 {
		t.Fatalf("saved: %q, %d trades", at, len(trades))
	}
	if trades[0].Date != "2023-04-09" || trades[3].Date != "2023-04-20" {
		t.Fatalf("not by date: %+v", trades)
	}
	if !trades[1].Canceled || trades[1].CanceledOn != "2023-04-27" || trades[0].RegisteredOn != "2023-06-01" {
		t.Fatalf("fields lost: %+v", trades[:2])
	}

	// Other months are untouched by a replace.
	if _, err := s.ReplaceMonth(ctx, "11110", "202305", nil, t2); err != nil {
		t.Fatal(err)
	}
	if trades, _, _ := s.Month(ctx, "11110", "202304"); len(trades) != 4 {
		t.Fatalf("202304 changed by a 202305 replace: %d", len(trades))
	}
}
