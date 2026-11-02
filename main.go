package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/freekrpark/jipwatch/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "jipwatch:", err)
		os.Exit(1)
	}
}

func run() error {
	path, err := dbPath()
	if err != nil {
		return err
	}
	st, err := store.Open(context.Background(), path)
	if err != nil {
		return err
	}
	defer st.Close()

	app := NewApp(st)
	return wails.Run(&options.App{
		Title:  "jipwatch",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})
}

// dbPath is %AppData%\jipwatch\jipwatch.db on Windows: per user, and
// outside any repository so nothing in it can be committed by mistake.
func dbPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "jipwatch")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "jipwatch.db"), nil
}
