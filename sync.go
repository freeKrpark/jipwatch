package main

import (
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
)

const (
	historyFrom  = "201601"
	recentMonths = 3
)

func syncPlan(fetched map[string]string, now time.Time) []string {
	start, _ := time.Parse("200601", historyFrom)
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	recent := end.AddDate(0, -(recentMonths - 1), 0)
	var plan []string
	for m := end; !m.Before(start); m = m.AddDate(0, -1, 0) {
		ym := m.Format("200601")
		if _, ok := fetched[ym]; !ok || !m.Before(recent) {
			plan = append(plan, ym)
		}
	}
	return plan
}

type Coverage struct {
	From    string `json:"from"`
	Fetched int    `json:"fetched"`
	Total   int    `json:"total"`
}

func coverage(fetched map[string]string, now time.Time) Coverage {
	all := syncPlan(map[string]string{}, now)
	c := Coverage{From: historyFrom, Total: len(all)}
	for _, ym := range all {
		if _, ok := fetched[ym]; ok {
			c.Fetched++
		}
	}
	return c
}

type WatchItem struct {
	AptSeq string       `json:"aptSeq"`
	AptNm  string       `json:"aptNm"`
	SggCd  string       `json:"sggCd"`
	UmdNm  string       `json:"umdNm"`
	Sales  int          `json:"sales"`
	Latest *molit.Trade `json:"latest"`
}

func watchItem(aptSeq, aptNm, sgg, umd string, history []molit.Trade) WatchItem {
	w := WatchItem{AptSeq: aptSeq, AptNm: aptNm, SggCd: sgg, UmdNm: umd}
	for i := range history {
		if history[i].Canceled {
			continue
		}
		w.Sales++
		w.Latest = &history[i]
	}
	return w
}

type SyncResult struct {
	Months        int    `json:"months"`
	Added         int    `json:"added"`
	NewlyCanceled int    `json:"newlyCanceled"`
	Error         string `json:"error"`
	RentMonths    int    `json:"rentMonths"`
	RentError     string `json:"rentError"`
}

type SyncProgress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Label string `json:"label"`
}

func ymLabel(ym string) string {
	t, err := time.Parse("200601", ym)
	if err != nil {
		return ym
	}
	return t.Format("2006년 1월")
}
