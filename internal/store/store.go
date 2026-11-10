package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // pure Go, so Windows builds need no C compiler
)

var migrations = []string{
	`CREATE TABLE settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
	// A district-month is fetched and replaced as a whole; fetches records when.
	`CREATE TABLE fetches (
		sgg_cd     TEXT NOT NULL,
		ym         TEXT NOT NULL,
		fetched_at TEXT NOT NULL,
		PRIMARY KEY (sgg_cd, ym)
	)`,
	`CREATE TABLE trades (
		sgg_cd        TEXT    NOT NULL,
		ym            TEXT    NOT NULL,
		apt_seq       TEXT    NOT NULL,
		apt_nm        TEXT    NOT NULL,
		umd_nm        TEXT    NOT NULL,
		jibun         TEXT    NOT NULL,
		area          REAL    NOT NULL,
		floor         INTEGER NOT NULL,
		price         INTEGER NOT NULL,
		date          TEXT    NOT NULL,
		direct        INTEGER NOT NULL,
		canceled      INTEGER NOT NULL,
		canceled_on   TEXT    NOT NULL,
		registered_on TEXT    NOT NULL,
		build_year    INTEGER NOT NULL,
		first_seen    TEXT    NOT NULL
	)`,
	`CREATE INDEX trades_by_month ON trades (sgg_cd, ym)`,
	`CREATE INDEX trades_by_complex ON trades (apt_seq, date)`,
	`CREATE TABLE watchlist (
		apt_seq  TEXT PRIMARY KEY,
		apt_nm   TEXT NOT NULL,
		sgg_cd   TEXT NOT NULL,
		umd_nm   TEXT NOT NULL,
		added_at TEXT NOT NULL
	)`,
	`CREATE TABLE rent_fetches (
		sgg_cd     TEXT NOT NULL,
		ym         TEXT NOT NULL,
		fetched_at TEXT NOT NULL,
		PRIMARY KEY (sgg_cd, ym)
	)`,
	`CREATE TABLE rents (
		sgg_cd        TEXT    NOT NULL,
		ym            TEXT    NOT NULL,
		apt_seq       TEXT    NOT NULL,
		apt_nm        TEXT    NOT NULL,
		umd_nm        TEXT    NOT NULL,
		jibun         TEXT    NOT NULL,
		area          REAL    NOT NULL,
		floor         INTEGER NOT NULL,
		deposit       INTEGER NOT NULL,
		monthly       INTEGER NOT NULL,
		date          TEXT    NOT NULL,
		kind          TEXT    NOT NULL,
		term          TEXT    NOT NULL,
		pre_deposit   INTEGER NOT NULL,
		pre_monthly   INTEGER NOT NULL,
		renewal_right INTEGER NOT NULL,
		build_year    INTEGER NOT NULL
	)`,
	`CREATE INDEX rents_by_month ON rents (sgg_cd, ym)`,
	`CREATE INDEX rents_by_complex ON rents (apt_seq, date)`,
	// ECOS fetch tracking: one row per kind ("m2" or "rate").
	`CREATE TABLE ecos_fetches (
		kind       TEXT PRIMARY KEY,
		fetched_at TEXT NOT NULL
	)`,
	// Monthly macro data from ECOS (M2 in 조원, base rate in %).
	`CREATE TABLE ecos_data (
		kind  TEXT NOT NULL,
		ym    TEXT NOT NULL,
		value REAL NOT NULL,
		PRIMARY KEY (kind, ym)
	)`,
}

type Store struct {
	db *sql.DB
}

// Open opens or creates the database at path and brings its schema up to date.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	// One connection: a desktop app has one user, and SQLite has one writer.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > len(migrations) {
		return fmt.Errorf("database is schema version %d, newer than this program (%d)", version, len(migrations))
	}
	for v := version; v < len(migrations); v++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[v]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", v+1, err)
		}
		// user_version lives in the file header and changes with the transaction.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", v+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Setting returns the value saved under key, or "" if there is none.
func (s *Store) Setting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetSetting saves value under key, replacing any earlier value.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
