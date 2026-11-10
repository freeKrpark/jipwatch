package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/freekrpark/jipwatch/internal/ecos"
	"github.com/freekrpark/jipwatch/internal/molit"
	"github.com/freekrpark/jipwatch/internal/store"
)

// settingAPIKey is where the data.go.kr key is saved. The key never goes
// back to the frontend; only Go code reads it.
const (
	settingAPIKey  = "molit.api_key"
	settingEcosKey = "ecos.api_key"
)

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

func (a *App) fetchRents(c *molit.Client, sgg, ym string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, time.Minute)
	defer cancel()
	fresh, err := c.Rents(ctx, sgg, ym)
	if err != nil {
		return err
	}
	return a.store.ReplaceRentMonth(a.ctx, sgg, ym, fresh, time.Now().In(kst))
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
// since historyFrom that was never fetched and the last few months again:
// sales first, then 전월세. It sends a "sync" event with SyncProgress before
// each month. Each kind stops at its first failure; the months fetched so far
// stay saved, and the next sync picks up the rest. A failure in rents, such
// as the rent API not yet applied for, leaves sales alone.
func (a *App) SyncWatched() (SyncResult, error) {
	if !a.syncing.CompareAndSwap(false, true) {
		return SyncResult{}, errors.New("이미 받는 중이에요")
	}
	defer a.syncing.Store(false)

	var r SyncResult

	// ECOS (global, once): fetch M2 and base rate before per-district jobs.
	if ec := a.ecosClient(); ec != nil {
		a.emit("sync", SyncProgress{Label: "M2·금리"})
		if err := a.fetchEcos(ec); err != nil {
			r.EcosError = fmt.Sprintf("ECOS: %v", err)
		}
	}

	list, err := a.store.Watchlist(a.ctx)
	if err != nil {
		return SyncResult{}, err
	}
	type job struct {
		sgg, ym string
		rent    bool
	}
	var jobs, rentJobs []job
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
			jobs = append(jobs, job{w.SggCd, ym, false})
		}
		rentFetched, err := a.store.RentFetched(a.ctx, w.SggCd)
		if err != nil {
			return SyncResult{}, err
		}
		for _, ym := range syncPlan(rentFetched, now) {
			rentJobs = append(rentJobs, job{w.SggCd, ym, true})
		}
	}
	jobs = append(jobs, rentJobs...)
	if len(jobs) == 0 {
		return r, nil
	}
	c, err := a.client()
	if err != nil {
		return SyncResult{}, err
	}

	done := 0
	for i, j := range jobs {
		label := districtName(j.sgg) + " " + ymLabel(j.ym)
		if j.rent {
			if r.Error != "" {
				break // rents would fail the same way
			}
			a.emit("sync", SyncProgress{Done: i, Total: len(jobs), Label: "전월세 · " + label})
			if err := a.fetchRents(c, j.sgg, j.ym); err != nil {
				r.RentError = rentError(label, err)
				break
			}
			r.RentMonths++
			done++
			continue
		}
		if r.Error != "" {
			continue
		}
		a.emit("sync", SyncProgress{Done: i, Total: len(jobs), Label: label})
		ch, err := a.fetch(c, j.sgg, j.ym)
		if err != nil {
			r.Error = fmt.Sprintf("%s에서 멈췄어요: %v", label, err)
			continue
		}
		r.Months++
		done++
		if !ch.First {
			r.Added += len(ch.Added)
			r.NewlyCanceled += len(ch.Canceled)
		}
	}
	a.emit("sync", SyncProgress{Done: done, Total: len(jobs)})
	return r, nil
}

// rentError says why rents stopped, pointing at the 활용신청 when the key is
// not registered for the rent API.
func rentError(label string, err error) string {
	var apiErr *molit.APIError
	if errors.As(err, &apiErr) && apiErr.Code == "30" {
		return "전월세는 못 받았어요. 공공데이터포털에서 '아파트 전월세 실거래가 자료' 활용신청이 반영됐는지 확인해 주세요"
	}
	return fmt.Sprintf("전월세는 %s에서 멈췄어요: %v", label, err)
}

func (a *App) Coverage(sgg string) (Coverage, error) {
	fetched, err := a.store.Fetched(a.ctx, sgg)
	if err != nil {
		return Coverage{}, err
	}
	return coverage(fetched, time.Now().In(kst)), nil
}

func (a *App) ComplexAnalysis(sgg, aptSeq string, area int) (Analysis, error) {
	history, err := a.store.ComplexHistory(a.ctx, sgg, aptSeq)
	if err != nil {
		return Analysis{}, err
	}
	now := time.Now().In(kst)
	an := analyse(history, area, now)
	district, err := a.store.DistrictTrades(a.ctx, sgg,
		float64(an.Area-bandHalf), float64(an.Area+bandHalf), aptSeq)
	if err != nil {
		return Analysis{}, err
	}
	an.Relative = relative(an, district, now)
	rents, err := a.store.ComplexRents(a.ctx, sgg, aptSeq)
	if err != nil {
		return Analysis{}, err
	}
	an.Jeonse = jeonse(an, rents, now)

	m2data, err := a.store.EcosData(a.ctx, "m2")
	if err != nil {
		return Analysis{}, err
	}
	rateData, err := a.store.EcosData(a.ctx, "rate")
	if err != nil {
		return Analysis{}, err
	}
	an.M2 = m2analysis(an.Trend, m2data, rateData)
	return an, nil
}

// --- ECOS (한국은행 경제통계) -------------------------------------------

// HasEcosKey reports whether an ECOS key has been saved.
func (a *App) HasEcosKey() (bool, error) {
	key, err := a.store.Setting(a.ctx, settingEcosKey)
	return key != "", err
}

// SaveEcosKey validates key against the ECOS API and saves it.
func (a *App) SaveEcosKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("ECOS API 키를 입력해 주세요")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	if err := ecos.NewClient(key).Ping(ctx); err != nil {
		var apiErr *ecos.APIError
		if errors.As(err, &apiErr) {
			return fmt.Errorf("ECOS 키 확인 실패: %s", apiErr.Message)
		}
		return fmt.Errorf("ECOS 키를 확인하지 못했어요: %w", err)
	}
	return a.store.SetSetting(a.ctx, settingEcosKey, key)
}

func (a *App) ecosClient() *ecos.Client {
	key, err := a.store.Setting(a.ctx, settingEcosKey)
	if err != nil || key == "" {
		return nil
	}
	return ecos.NewClient(key)
}

func (a *App) fetchEcos(ec *ecos.Client) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	now := time.Now().In(kst)
	endYM := now.Format("200601")

	m2rows, err := ec.M2(ctx, historyFrom, endYM)
	if err != nil {
		return fmt.Errorf("M2: %w", err)
	}
	var m2points []store.EcosPoint
	for _, r := range m2rows {
		if v, ok := ecos.ParseValue(r.Value); ok {
			m2points = append(m2points, store.EcosPoint{YM: r.Time, Value: v})
		}
	}
	if err := a.store.ReplaceEcos(a.ctx, "m2", m2points, now); err != nil {
		return err
	}

	rateRows, err := ec.BaseRate(ctx, historyFrom, endYM)
	if err != nil {
		return fmt.Errorf("기준금리: %w", err)
	}
	var ratePoints []store.EcosPoint
	for _, r := range rateRows {
		if v, ok := ecos.ParseValue(r.Value); ok {
			ratePoints = append(ratePoints, store.EcosPoint{YM: r.Time, Value: v})
		}
	}
	return a.store.ReplaceEcos(a.ctx, "rate", ratePoints, now)
}

type BuyRentStart struct {
	Input  BuyRentInput `json:"input"`
	Growth Growth       `json:"growth"`
}

func buyRentKey(aptSeq string, area int) string {
	return fmt.Sprintf("buyrent:%s:%d", aptSeq, area)
}

const buyRentLast = "buyrent:last"

func (a *App) BuyRentStart(sgg, aptSeq string, area int) (BuyRentStart, error) {
	history, err := a.store.ComplexHistory(a.ctx, sgg, aptSeq)
	if err != nil {
		return BuyRentStart{}, err
	}
	now := time.Now().In(kst)
	an := analyse(history, area, now)
	rents, err := a.store.ComplexRents(a.ctx, sgg, aptSeq)
	if err != nil {
		return BuyRentStart{}, err
	}
	recentJeonse := jeonse(an, rents, now).RecentJeonse
	in := BuyRentInput{Price: an.Summary.RecentMedian, Area: an.Area,
		LoanRate: 4.5, JeonseRate: 4.1, OppRate: 3, MaintRate: 0.5, Years: 10}
	if last, err := a.store.Setting(a.ctx, buyRentLast); err == nil && last != "" {
		var l BuyRentInput
		if json.Unmarshal([]byte(last), &l) == nil {
			in.Cash, in.LoanRate, in.OppRate, in.MaintRate, in.Years = l.Cash, l.LoanRate, l.OppRate, l.MaintRate, l.Years
			if l.JeonseRate > 0 {
				in.JeonseRate = l.JeonseRate
			}
		}
	}
	if saved, err := a.store.Setting(a.ctx, buyRentKey(aptSeq, an.Area)); err == nil && saved != "" {
		json.Unmarshal([]byte(saved), &in)
	}
	if in.Jeonse == 0 {
		in.Jeonse = recentJeonse
	}
	return BuyRentStart{Input: in, Growth: growth(an.Trend)}, nil
}

func (a *App) BuyRent(aptSeq string, in BuyRentInput) (BuyRentResult, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return BuyRentResult{}, err
	}
	if err := a.store.SetSetting(a.ctx, buyRentKey(aptSeq, in.Area), string(b)); err != nil {
		return BuyRentResult{}, err
	}
	if err := a.store.SetSetting(a.ctx, buyRentLast, string(b)); err != nil {
		return BuyRentResult{}, err
	}
	return buyRent(in), nil
}
