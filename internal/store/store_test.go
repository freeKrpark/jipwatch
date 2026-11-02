package store

import (
	"context"
	"path/filepath"
	"testing"
)

func open(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSettings(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "test.db"))

	if v, err := s.Setting(ctx, "missing"); err != nil || v != "" {
		t.Fatalf("missing setting: %q, %v", v, err)
	}
	for _, want := range []string{"first", "second"} {
		if err := s.SetSetting(ctx, "k", want); err != nil {
			t.Fatal(err)
		}
		if got, err := s.Setting(ctx, "k"); err != nil || got != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	}
}

func TestSettingsSurviveReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, "k", "v"); err != nil {
		t.Fatal(err)
	}
	s.Close()

	// Reopening runs migrate again; it must not touch an up-to-date schema.
	s = open(t, path)
	if got, _ := s.Setting(ctx, "k"); got != "v" {
		t.Fatalf("after reopen: %q", got)
	}
	var version int
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != len(migrations) {
		t.Fatalf("user_version %d, want %d", version, len(migrations))
	}
}

// An older program must not run against a database a newer one has upgraded.
func TestRefusesNewerSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	s := open(t, path)
	s.db.Exec("PRAGMA user_version = 99")
	s.Close()
	if _, err := Open(ctx, path); err == nil {
		t.Fatal("want error")
	}
}
