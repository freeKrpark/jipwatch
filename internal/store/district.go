package store

import (
	"context"

	"github.com/freekrpark/jipwatch/internal/molit"
)

func (s *Store) DistrictTrades(ctx context.Context, sgg string, areaMin, areaMax float64, skip string) ([]molit.Trade, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+tradeColumns+` FROM trades
		 WHERE sgg_cd = ? AND area >= ? AND area < ? AND apt_seq <> ?
		 ORDER BY date, rowid`, sgg, areaMin, areaMax, skip)
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
