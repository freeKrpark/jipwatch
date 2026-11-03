package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func (s *Store) ReplaceRentMonth(ctx context.Context, sgg, ym string, fresh []molit.Rent, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM rents WHERE sgg_cd = ? AND ym = ?", sgg, ym); err != nil {
		return err
	}
	insert, err := tx.PrepareContext(ctx, `INSERT INTO rents (
		sgg_cd, ym, apt_seq, apt_nm, umd_nm, jibun, area, floor, deposit, monthly, date,
		kind, term, pre_deposit, pre_monthly, renewal_right, build_year
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insert.Close()
	for _, r := range fresh {
		if _, err := insert.ExecContext(ctx, sgg, ym, r.AptSeq, r.AptNm, r.UmdNm, r.Jibun, r.Area, r.Floor,
			r.Deposit, r.Monthly, r.Date, r.Kind, r.Term, r.PreDeposit, r.PreMonthly, r.RenewalRight,
			r.BuildYear); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO rent_fetches (sgg_cd, ym, fetched_at) VALUES (?, ?, ?)
		 ON CONFLICT (sgg_cd, ym) DO UPDATE SET fetched_at = excluded.fetched_at`,
		sgg, ym, now.Format("2006-01-02 15:04")); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RentFetched(ctx context.Context, sgg string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT ym, fetched_at FROM rent_fetches WHERE sgg_cd = ?", sgg)
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

func (s *Store) ComplexRents(ctx context.Context, sgg, aptSeq string) ([]molit.Rent, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+rentColumns+" FROM rents WHERE apt_seq = ? ORDER BY date, rowid", aptSeq)
	if err != nil {
		return nil, err
	}
	return scanRents(rows, sgg)
}

const rentColumns = `apt_seq, apt_nm, umd_nm, jibun, area, floor, deposit, monthly, date,
	kind, term, pre_deposit, pre_monthly, renewal_right, build_year`

func scanRents(rows *sql.Rows, sgg string) ([]molit.Rent, error) {
	defer rows.Close()
	out := []molit.Rent{}
	for rows.Next() {
		r := molit.Rent{SggCd: sgg}
		if err := rows.Scan(&r.AptSeq, &r.AptNm, &r.UmdNm, &r.Jibun, &r.Area, &r.Floor, &r.Deposit, &r.Monthly,
			&r.Date, &r.Kind, &r.Term, &r.PreDeposit, &r.PreMonthly, &r.RenewalRight, &r.BuildYear); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
