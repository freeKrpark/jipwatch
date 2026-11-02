package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func TestDistrictTrades(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	mine := sale("09", 90000) // 11110-0001, 84.95㎡
	other := sale("10", 95000)
	other.AptSeq = "11110-0002"
	small := sale("11", 60000)
	small.AptSeq, small.Area = "11110-0003", 59.98
	if _, err := s.ReplaceMonth(ctx, "11110", "202304", []molit.Trade{mine, other, small}, now); err != nil {
		t.Fatal(err)
	}

	got, err := s.DistrictTrades(ctx, "11110", 79, 89, "11110-0001")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AptSeq != "11110-0002" {
		t.Fatalf("want only the other 84㎡ complex, got %+v", got)
	}
}
