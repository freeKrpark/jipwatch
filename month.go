package main

import (
	"errors"
	"sort"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

type MonthSummary struct {
	Total     int            `json:"total"`
	Canceled  int            `json:"canceled"`
	Direct    int            `json:"direct"`
	Complexes []ComplexCount `json:"complexes"`

	FetchedAt     string `json:"fetchedAt"`
	Compared      bool   `json:"compared"`
	Added         int    `json:"added"`
	NewlyCanceled int    `json:"newlyCanceled"`
}

type ComplexCount struct {
	AptSeq    string `json:"aptSeq"`
	AptNm     string `json:"aptNm"`
	UmdNm     string `json:"umdNm"`
	BuildYear int    `json:"buildYear"`
	Sales     int    `json:"sales"`
}

func checkQuery(sgg, month string) error {
	if len(sgg) != 5 {
		return errors.New("구 코드는 5자리 숫자예요 (예: 11110)")
	}
	if _, err := time.Parse("200601", month); err != nil {
		return errors.New("월은 YYYYMM 형식이에요 (예: 202304)")
	}
	return nil
}

func summarize(trades []molit.Trade) MonthSummary {
	s := MonthSummary{Total: len(trades), Complexes: []ComplexCount{}}
	index := map[string]int{}
	for _, t := range trades {
		if t.Canceled {
			s.Canceled++
		}
		if t.Direct {
			s.Direct++
		}
		i, ok := index[t.AptSeq]
		if !ok {
			i = len(s.Complexes)
			index[t.AptSeq] = i
			s.Complexes = append(s.Complexes, ComplexCount{
				AptSeq: t.AptSeq, AptNm: t.AptNm, UmdNm: t.UmdNm, BuildYear: t.BuildYear,
			})
		}
		s.Complexes[i].Sales++
	}
	sort.Slice(s.Complexes, func(i, j int) bool {
		a, b := s.Complexes[i], s.Complexes[j]
		if a.Sales != b.Sales {
			return a.Sales > b.Sales
		}
		return a.AptSeq < b.AptSeq
	})
	return s
}

func complexTrades(trades []molit.Trade, aptSeq string) []molit.Trade {
	mine := []molit.Trade{}
	for _, t := range trades {
		if t.AptSeq == aptSeq {
			mine = append(mine, t)
		}
	}
	sort.SliceStable(mine, func(i, j int) bool { return mine[i].Date < mine[j].Date })
	return mine
}
