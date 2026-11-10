package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type EcosPoint struct {
	YM    string
	Value float64
}

func (s *Store) ReplaceEcos(ctx context.Context, kind string, points []EcosPoint, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM ecos_data WHERE kind = ?", kind); err != nil {
		return err
	}
	insert, err := tx.PrepareContext(ctx, "INSERT INTO ecos_data (kind, ym, value) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer insert.Close()
	for _, p := range points {
		if _, err := insert.ExecContext(ctx, kind, p.YM, p.Value); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO ecos_fetches (kind, fetched_at) VALUES (?, ?)
		 ON CONFLICT (kind) DO UPDATE SET fetched_at = excluded.fetched_at`,
		kind, now.Format("2006-01-02 15:04")); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) EcosData(ctx context.Context, kind string) ([]EcosPoint, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT ym, value FROM ecos_data WHERE kind = ? ORDER BY ym", kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EcosPoint
	for rows.Next() {
		var p EcosPoint
		if err := rows.Scan(&p.YM, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) EcosFetchedAt(ctx context.Context, kind string) (string, error) {
	var at string
	err := s.db.QueryRowContext(ctx,
		"SELECT fetched_at FROM ecos_fetches WHERE kind = ?", kind).Scan(&at)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return at, err
}
