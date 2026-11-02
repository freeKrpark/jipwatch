package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func TestWatchlist(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)

	if list, err := s.Watchlist(ctx); err != nil || len(list) != 0 || list == nil {
		t.Fatalf("empty watchlist: %v %v", list, err)
	}
	a := Watched{AptSeq: "11110-0001", AptNm: "첫째아파트", SggCd: "11110", UmdNm: "테스트동"}
	b := Watched{AptSeq: "11110-0002", AptNm: "둘째아파트", SggCd: "11110", UmdNm: "테스트동"}
	for _, w := range []Watched{a, b, a} { // adding a twice keeps one
		if err := s.Watch(ctx, w, now); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Minute)
	}
	list, _ := s.Watchlist(ctx)
	if len(list) != 2 || list[0].AptSeq != a.AptSeq || list[0].AddedAt != "2026-11-02 09:00" {
		t.Fatalf("watchlist: %+v", list)
	}
	if err := s.Unwatch(ctx, a.AptSeq); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.IsWatched(ctx, a.AptSeq); ok {
		t.Fatal("still watched")
	}
	if ok, _ := s.IsWatched(ctx, b.AptSeq); !ok {
		t.Fatal("b dropped")
	}
}

func TestComplexHistoryAcrossMonths(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	may := sale("01", 91000)
	may.Date = "2023-05-01"
	other := sale("02", 50000)
	other.AptSeq = "11110-0002"
	s.ReplaceMonth(ctx, "11110", "202305", []molit.Trade{may}, now)
	s.ReplaceMonth(ctx, "11110", "202304", []molit.Trade{sale("09", 90000), other}, now)

	got, err := s.ComplexHistory(ctx, "11110", "11110-0001")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Date != "2023-04-09" || got[1].Date != "2023-05-01" {
		t.Fatalf("history: %+v", got)
	}
	fetched, _ := s.Fetched(ctx, "11110")
	if len(fetched) != 2 || fetched["202304"] != "2026-11-02 09:00" {
		t.Fatalf("fetched: %v", fetched)
	}
}
