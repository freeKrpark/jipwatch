package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestEcosRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	points := []EcosPoint{
		{"202301", 3812.5},
		{"202302", 3830.1},
	}
	if err := s.ReplaceEcos(ctx, "m2", points, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.EcosData(ctx, "m2")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Value != 3812.5 || got[1].YM != "202302" {
		t.Fatalf("round-trip: %+v", got)
	}
	at, err := s.EcosFetchedAt(ctx, "m2")
	if err != nil || at == "" {
		t.Fatalf("fetched_at %q, %v", at, err)
	}
}

func TestEcosReplaceDeletesOld(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	if err := s.ReplaceEcos(ctx, "m2", []EcosPoint{{"202301", 100}, {"202302", 200}}, now); err != nil {
		t.Fatal(err)
	}
	// Replace with fewer rows.
	if err := s.ReplaceEcos(ctx, "m2", []EcosPoint{{"202301", 150}}, now); err != nil {
		t.Fatal(err)
	}
	got, _ := s.EcosData(ctx, "m2")
	if len(got) != 1 || got[0].Value != 150 {
		t.Fatalf("after replace: %+v", got)
	}
}

func TestEcosKindsAreIndependent(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	if err := s.ReplaceEcos(ctx, "m2", []EcosPoint{{"202301", 3000}}, now); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceEcos(ctx, "rate", []EcosPoint{{"202301", 3.5}}, now); err != nil {
		t.Fatal(err)
	}
	m2, _ := s.EcosData(ctx, "m2")
	rate, _ := s.EcosData(ctx, "rate")
	if len(m2) != 1 || len(rate) != 1 {
		t.Fatalf("m2=%d rate=%d", len(m2), len(rate))
	}
	if m2[0].Value != 3000 || rate[0].Value != 3.5 {
		t.Fatalf("m2=%+v rate=%+v", m2, rate)
	}
}

func TestEcosEmpty(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))

	got, err := s.EcosData(ctx, "m2")
	if err != nil || got != nil {
		t.Fatalf("empty: %+v, %v", got, err)
	}
	at, err := s.EcosFetchedAt(ctx, "m2")
	if err != nil || at != "" {
		t.Fatalf("empty fetched_at: %q, %v", at, err)
	}
}
