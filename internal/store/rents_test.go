package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func lease(apt, day string, deposit, monthly int, kind string) molit.Rent {
	return molit.Rent{AptSeq: apt, AptNm: "첫째아파트", SggCd: "11110", UmdNm: "테스트동", Jibun: "1",
		Area: 84.95, Floor: 9, Deposit: deposit, Monthly: monthly, Date: "2025-09-" + day,
		Kind: kind, Term: "25.11~27.11", BuildYear: 2011}
}

func TestRentsReplaceAndRead(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	now := time.Date(2025, 10, 1, 9, 0, 0, 0, time.UTC)

	renewal := lease("11110-0001", "02", 63000, 0, "갱신")
	renewal.PreDeposit, renewal.RenewalRight = 60000, true
	first := []molit.Rent{
		lease("11110-0001", "08", 60000, 0, "신규"),
		renewal,
		lease("11110-0002", "10", 20000, 150, ""),
	}
	if err := s.ReplaceRentMonth(ctx, "11110", "202509", first, now); err != nil {
		t.Fatal(err)
	}
	// A refetch replaces the month: a late report arrives.
	again := append(first, lease("11110-0001", "20", 61000, 0, "신규"))
	if err := s.ReplaceRentMonth(ctx, "11110", "202509", again, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := s.ComplexRents(ctx, "11110", "11110-0001")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Date != "2025-09-02" || got[2].Deposit != 61000 {
		t.Fatalf("complex rents: %+v", got)
	}
	if r := got[0]; r.Kind != "갱신" || r.PreDeposit != 60000 || !r.RenewalRight || r.SggCd != "11110" {
		t.Fatalf("renewal came back as %+v", r)
	}

	fetched, err := s.RentFetched(ctx, "11110")
	if err != nil || fetched["202509"] != "2025-10-01 10:00" {
		t.Fatalf("rent fetches: %v, %v", fetched, err)
	}
	// Sales fetches are kept apart.
	if sales, _ := s.Fetched(ctx, "11110"); len(sales) != 0 {
		t.Fatalf("sales fetches: %v", sales)
	}
}

func TestComplexRentsEmpty(t *testing.T) {
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	got, err := s.ComplexRents(context.Background(), "11110", "11110-0009")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}
