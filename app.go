package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/freekrpark/jipwatch/internal/molit"
	"github.com/freekrpark/jipwatch/internal/store"
)

// settingAPIKey is where the data.go.kr key is saved. The key never goes
// back to the frontend; only Go code reads it.
const settingAPIKey = "molit.api_key"

var kst = time.FixedZone("KST", 9*60*60)

// App holds the methods the frontend can call.
type App struct {
	ctx     context.Context
	store   *store.Store
	emit    func(event string, data any)
	mu      sync.Mutex  // one API fetch at a time
	syncing atomic.Bool // one sync at a time
}

func NewApp(st *store.Store) *App {
	return &App{store: st, emit: func(string, any) {}}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.emit = func(event string, data any) { wailsruntime.EventsEmit(ctx, event, data) }
}

// --- API key ----------------------------------------------------------------

// HasAPIKey reports whether a key has been saved.
func (a *App) HasAPIKey() (bool, error) {
	key, err := a.store.Setting(a.ctx, settingAPIKey)
	return key != "", err
}

// SaveAPIKey tries key against the API and saves it only if it works.
func (a *App) SaveAPIKey(key string) error {
	key = molit.NormalizeKey(key)
	if key == "" {
		return errors.New("API 키를 입력해 주세요")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	if err := molit.NewClient(key).Ping(ctx); err != nil {
		var apiErr *molit.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "30" {
			return errors.New("등록되지 않은 키예요. 마이페이지의 Decoding 키인지, " +
				"'아파트 매매 실거래가 상세 자료' 활용신청이 반영됐는지 확인해 주세요")
		}
		return fmt.Errorf("키를 확인하지 못했어요: %w", err)
	}
	return a.store.SetSetting(a.ctx, settingAPIKey, key)
}

func (a *App) client() (*molit.Client, error) {
	key, err := a.store.Setting(a.ctx, settingAPIKey)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, errors.New("API 키를 먼저 설정해 주세요")
	}
	return molit.NewClient(key), nil
}

// fetch gets one district-month from the API and saves it in place of the
// saved copy.
func (a *App) fetch(c *molit.Client, sgg, ym string) (store.Change, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, time.Minute)
	defer cancel()
	fresh, err := c.Month(ctx, sgg, ym)
	if err != nil {
		return store.Change{}, err
	}
	return a.store.ReplaceMonth(a.ctx, sgg, ym, fresh, time.Now().In(kst))
}

// --- one month --------------------------------------------------------------

// DefaultMonth is last month in Korea, "200601" layout: the latest month
// with a fair share of its sales reported.
func (a *App) DefaultMonth() string {
	return time.Now().In(kst).AddDate(0, -1, 0).Format("200601")
}

// Month lists the complexes with sales in district sgg in month, from the
// saved copy if there is one and from the API otherwise.
func (a *App) Month(sgg, month string) (MonthSummary, error) {
	return a.summary(sgg, month, false)
}

// RefreshMonth fetches the month from the API again, saves it in place of
// the old copy, and counts what changed.
func (a *App) RefreshMonth(sgg, month string) (MonthSummary, error) {
	return a.summary(sgg, month, true)
}

// ComplexTrades lists one complex's sales in district sgg in month.
func (a *App) ComplexTrades(sgg, month, aptSeq string) ([]molit.Trade, error) {
	trades, _, _, err := a.trades(sgg, month, false)
	if err != nil {
		return nil, err
	}
	return complexTrades(trades, aptSeq), nil
}

func (a *App) summary(sgg, month string, refresh bool) (MonthSummary, error) {
	trades, fetchedAt, change, err := a.trades(sgg, month, refresh)
	if err != nil {
		return MonthSummary{}, err
	}
	s := summarize(trades)
	s.FetchedAt = fetchedAt
	if change != nil && !change.First {
		s.Compared, s.Added, s.NewlyCanceled = true, len(change.Added), len(change.Canceled)
	}
	return s, nil
}

// trades returns the saved copy of a month unless refresh is set or there is
// none; then it fetches the month, saves it, and returns what changed.
func (a *App) trades(sgg, month string, refresh bool) ([]molit.Trade, string, *store.Change, error) {
	if err := checkQuery(sgg, month); err != nil {
		return nil, "", nil, err
	}
	if !refresh {
		saved, fetchedAt, err := a.store.Month(a.ctx, sgg, month)
		if err != nil || fetchedAt != "" {
			return saved, fetchedAt, nil, err
		}
	}
	c, err := a.client()
	if err != nil {
		return nil, "", nil, err
	}
	change, err := a.fetch(c, sgg, month)
	if err != nil {
		return nil, "", nil, err
	}
	saved, fetchedAt, err := a.store.Month(a.ctx, sgg, month)
	return saved, fetchedAt, &change, err
}

// --- watchlist --------------------------------------------------------------

// Watch adds a complex to the watchlist. Call SyncWatched afterwards to
// fetch its district's history.
func (a *App) Watch(aptSeq, aptNm, sgg, umdNm string) error {
	return a.store.Watch(a.ctx, store.Watched{AptSeq: aptSeq, AptNm: aptNm, SggCd: sgg, UmdNm: umdNm}, time.Now().In(kst))
}

// Unwatch removes a complex from the watchlist; its saved sales stay.
func (a *App) Unwatch(aptSeq string) error {
	return a.store.Unwatch(a.ctx, aptSeq)
}

// IsWatched reports whether a complex is on the watchlist.
func (a *App) IsWatched(aptSeq string) (bool, error) {
	return a.store.IsWatched(a.ctx, aptSeq)
}

// Watchlist lists the watched complexes with their saved history summarized.
func (a *App) Watchlist() ([]WatchItem, error) {
	list, err := a.store.Watchlist(a.ctx)
	if err != nil {
		return nil, err
	}
	items := make([]WatchItem, 0, len(list))
	for _, w := range list {
		history, err := a.store.ComplexHistory(a.ctx, w.SggCd, w.AptSeq)
		if err != nil {
			return nil, err
		}
		items = append(items, watchItem(w.AptSeq, w.AptNm, w.SggCd, w.UmdNm, history))
	}
	return items, nil
}

// ComplexHistory returns every saved sale of one complex by contract date.
func (a *App) ComplexHistory(sgg, aptSeq string) ([]molit.Trade, error) {
	trades, err := a.store.ComplexHistory(a.ctx, sgg, aptSeq)
	if trades == nil && err == nil {
		trades = []molit.Trade{}
	}
	return trades, err
}

// SyncWatched fetches, for every district with a watched complex, each month
// since historyFrom that was never fetched and the last few months again.
// It sends a "sync" event with SyncProgress before each month. It stops at
// the first failure; the months fetched so far stay saved, and the next sync
// picks up the rest.
func (a *App) SyncWatched() (SyncResult, error) {
	if !a.syncing.CompareAndSwap(false, true) {
		return SyncResult{}, errors.New("이미 받는 중이에요")
	}
	defer a.syncing.Store(false)

	list, err := a.store.Watchlist(a.ctx)
	if err != nil {
		return SyncResult{}, err
	}
	type job struct{ sgg, ym string }
	var jobs []job
	seen := map[string]bool{}
	now := time.Now().In(kst)
	for _, w := range list {
		if seen[w.SggCd] {
			continue
		}
		seen[w.SggCd] = true
		fetched, err := a.store.Fetched(a.ctx, w.SggCd)
		if err != nil {
			return SyncResult{}, err
		}
		for _, ym := range syncPlan(fetched, now) {
			jobs = append(jobs, job{w.SggCd, ym})
		}
	}
	if len(jobs) == 0 {
		return SyncResult{}, nil
	}
	c, err := a.client()
	if err != nil {
		return SyncResult{}, err
	}

	var r SyncResult
	for i, j := range jobs {
		a.emit("sync", SyncProgress{Done: i, Total: len(jobs), Label: j.sgg + " " + ymLabel(j.ym)})
		ch, err := a.fetch(c, j.sgg, j.ym)
		if err != nil {
			r.Error = fmt.Sprintf("%s %s에서 멈췄어요: %v", j.sgg, ymLabel(j.ym), err)
			break
		}
		r.Months++
		if !ch.First {
			r.Added += len(ch.Added)
			r.NewlyCanceled += len(ch.Canceled)
		}
	}
	a.emit("sync", SyncProgress{Done: r.Months, Total: len(jobs)})
	return r, nil
}
