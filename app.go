package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/freekrpark/jipwatch/internal/molit"
	"github.com/freekrpark/jipwatch/internal/store"
)

// settingAPIKey is where the data.go.kr key is saved. The key never goes
// back to the frontend; only Go code reads it.
const settingAPIKey = "molit.api_key"

// App holds the methods the frontend can call.
type App struct {
	ctx   context.Context
	store *store.Store
}

func NewApp(st *store.Store) *App {
	return &App{store: st}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// HasAPIKey reports whether a key has been saved.
func (a *App) HasAPIKey() (bool, error) {
	key, err := a.store.Setting(a.ctx, settingAPIKey)
	return key != "", err
}

// SaveAPIKey tries key against the API and saves it only if it works.
func (a *App) SaveAPIKey(key string) error {
	key = strings.TrimSpace(key)
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
