package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

type Change struct {
	First    bool          `json:"first"`
	Added    []molit.Trade `json:"added"`
	Canceled []molit.Trade `json:"canceled"`
}

type saved struct {
	molit.Trade
	firstSeen string
}

func saleKey(t molit.Trade) string {
	return fmt.Sprintf("%s|%s|%.3f|%d|%d|%t", t.AptSeq, t.Date, t.Area, t.Floor, t.Price, t.Direct)
}

func match(old []saved, fresh []molit.Trade, now string) ([]saved, Change) {
	pool := map[string][]string{} // key+canceled -> first-seen times not yet paired
	for _, o := range old {
		k := poolKey(o.Trade)
		pool[k] = append(pool[k], o.firstSeen)
	}
	var change Change
	rows := make([]saved, 0, len(fresh))
	for _, t := range fresh {
		k := poolKey(t)
		first := now
		if seen := pool[k]; len(seen) > 0 {
			first, pool[k] = seen[0], seen[1:]
		} else if t.Canceled {
			change.Canceled = append(change.Canceled, t)
		} else {
			change.Added = append(change.Added, t)
		}
		rows = append(rows, saved{Trade: t, firstSeen: first})
	}
	return rows, change
}

func poolKey(t molit.Trade) string { return fmt.Sprintf("%s|%t", saleKey(t), t.Canceled) }

func (s *Store) Month(ctx context.Context, sgg, ym string) (trades []molit.Trade, fetchedAt string, err error) {
	err = s.db.QueryRowContext(ctx,
		"SELECT fetched_at FROM fetches WHERE sgg_cd = ? AND ym = ?", sgg, ym).Scan(&fetchedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, "", nil
		}
		return nil, "", err
	}
	rows, err := s.load(ctx, s.db, sgg, ym)
	if err != nil {
		return nil, "", err
	}
	trades = make([]molit.Trade, len(rows))
	for i, r := range rows {
		trades[i] = r.Trade
	}
	return trades, fetchedAt, nil
}

func (s *Store) ReplaceMonth(ctx context.Context, sgg, ym string, fresh []molit.Trade, now time.Time) (Change, error) {
	stamp := now.Format("2006-01-02 15:04")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Change{}, err
	}
	defer tx.Rollback()

	var fetchedBefore int
	if err := tx.QueryRowContext(ctx,
		"SELECT count(*) FROM fetches WHERE sgg_cd = ? AND ym = ?", sgg, ym).Scan(&fetchedBefore); err != nil {
		return Change{}, err
	}
	old, err := s.load(ctx, tx, sgg, ym)
	if err != nil {
		return Change{}, err
	}
	rows, change := match(old, fresh, stamp)
	if fetchedBefore == 0 {
		change = Change{First: true}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM trades WHERE sgg_cd = ? AND ym = ?", sgg, ym); err != nil {
		return Change{}, err
	}
	insert, err := tx.PrepareContext(ctx, `INSERT INTO trades (
		sgg_cd, ym, apt_seq, apt_nm, umd_nm, jibun, area, floor, price, date,
		direct, canceled, canceled_on, registered_on, build_year, first_seen
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return Change{}, err
	}
	defer insert.Close()
	for _, r := range rows {
		t := r.Trade
		if _, err := insert.ExecContext(ctx, sgg, ym, t.AptSeq, t.AptNm, t.UmdNm, t.Jibun, t.Area, t.Floor,
			t.Price, t.Date, t.Direct, t.Canceled, t.CanceledOn, t.RegisteredOn, t.BuildYear, r.firstSeen); err != nil {
			return Change{}, err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO fetches (sgg_cd, ym, fetched_at) VALUES (?, ?, ?)
		 ON CONFLICT (sgg_cd, ym) DO UPDATE SET fetched_at = excluded.fetched_at`, sgg, ym, stamp); err != nil {
		return Change{}, err
	}
	return change, tx.Commit()
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *Store) load(ctx context.Context, q querier, sgg, ym string) ([]saved, error) {
	rows, err := q.QueryContext(ctx,
		"SELECT "+tradeColumns+" FROM trades WHERE sgg_cd = ? AND ym = ? ORDER BY date, rowid", sgg, ym)
	if err != nil {
		return nil, err
	}
	return scanSaved(rows, sgg)
}

const tradeColumns = `apt_seq, apt_nm, umd_nm, jibun, area, floor, price, date,
	direct, canceled, canceled_on, registered_on, build_year, first_seen`

func scanSaved(rows *sql.Rows, sgg string) ([]saved, error) {
	defer rows.Close()
	var out []saved
	for rows.Next() {
		r := saved{Trade: molit.Trade{SggCd: sgg}}
		t := &r.Trade
		if err := rows.Scan(&t.AptSeq, &t.AptNm, &t.UmdNm, &t.Jibun, &t.Area, &t.Floor, &t.Price, &t.Date,
			&t.Direct, &t.Canceled, &t.CanceledOn, &t.RegisteredOn, &t.BuildYear, &r.firstSeen); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
