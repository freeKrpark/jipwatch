package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
	"github.com/freekrpark/jipwatch/internal/store"
)

// settingAPIKey is where the data.go.kr key is saved. The key never goes
// back to the frontend; only Go code reads it.
const settingAPIKey = "molit.api_key"

var kst = time.FixedZone("KST", 9*60*60)

// App holds the methods the frontend can call.
type App struct {
	ctx   context.Context
	store *store.Store
	mu    sync.Mutex // one fetch at a time
}

func NewApp(st *store.Store) *App {
	return &App{store: st}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

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
	a.mu.Lock()
	defer a.mu.Unlock()
	if !refresh {
		saved, fetchedAt, err := a.store.Month(a.ctx, sgg, month)
		if err != nil || fetchedAt != "" {
			return saved, fetchedAt, nil, err
		}
	}

	key, err := a.store.Setting(a.ctx, settingAPIKey)
	if err != nil {
		return nil, "", nil, err
	}
	if key == "" {
		return nil, "", nil, errors.New("API 키를 먼저 설정해 주세요")
	}
	ctx, cancel := context.WithTimeout(a.ctx, time.Minute)
	defer cancel()
	fresh, err := molit.NewClient(key).Month(ctx, sgg, month)
	if err != nil {
		return nil, "", nil, err
	}
	change, err := a.store.ReplaceMonth(a.ctx, sgg, month, fresh, time.Now().In(kst))
	if err != nil {
		return nil, "", nil, err
	}
	saved, fetchedAt, err := a.store.Month(a.ctx, sgg, month)
	return saved, fetchedAt, &change, err
}
