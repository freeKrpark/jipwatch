package store

import (
	"context"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

type Watched struct {
	AptSeq  string `json:"aptSeq"`
	AptNm   string `json:"aptNm"`
	SggCd   string `json:"sggCd"`
	UmdNm   string `json:"umdNm"`
	AddedAt string `json:"addedAt"`
}

func (s *Store) Watch(ctx context.Context, w Watched, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO watchlist (apt_seq, apt_nm, sgg_cd, umd_nm, added_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (apt_seq) DO NOTHING`,
		w.AptSeq, w.AptNm, w.SggCd, w.UmdNm, now.Format("2006-01-02 15:04"))
	return err
}

func (s *Store) Unwatch(ctx context.Context, aptSeq string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM watchlist WHERE apt_seq = ?", aptSeq)
	return err
}

func (s *Store) Watchlist(ctx context.Context) ([]Watched, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT apt_seq, apt_nm, sgg_cd, umd_nm, added_at FROM watchlist ORDER BY added_at, apt_seq")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Watched{}
	for rows.Next() {
		var w Watched
		if err := rows.Scan(&w.AptSeq, &w.AptNm, &w.SggCd, &w.UmdNm, &w.AddedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (s *Store) IsWatched(ctx context.Context, aptSeq string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM watchlist WHERE apt_seq = ?", aptSeq).Scan(&n)
	return n > 0, err
}

func (s *Store) Fetched(ctx context.Context, sgg string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT ym, fetched_at FROM fetches WHERE sgg_cd = ?", sgg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var ym, at string
		if err := rows.Scan(&ym, &at); err != nil {
			return nil, err
		}
		out[ym] = at
	}
	return out, rows.Err()
}

func (s *Store) ComplexHistory(ctx context.Context, sgg, aptSeq string) ([]molit.Trade, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+tradeColumns+" FROM trades WHERE apt_seq = ? ORDER BY date, rowid", aptSeq)
	if err != nil {
		return nil, err
	}
	all, err := scanSaved(rows, sgg)
	if err != nil {
		return nil, err
	}
	trades := make([]molit.Trade, len(all))
	for i, r := range all {
		trades[i] = r.Trade
	}
	return trades, nil
}
