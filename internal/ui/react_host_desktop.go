//go:build windows || (darwin && cgo)

package ui

import (
	"bytes"
	"clipare/internal/clipboard"
	"context"
	"errors"
	"fmt"
	"image"
	imgcolor "image/color"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// RunReactWithReady is the opt-in migration host. The existing Go GUI loop
// remains the single owner of sync, pairing, settings, history and updates.
func RunReactWithReady(parent context.Context, path string, log *slog.Logger, ready func() error, restored bool) error {
	assets, err := reactAssetDir()
	if err != nil {
		return err
	}
	backend, err := clipboard.New()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var app *application.App
	var tray *application.SystemTray
	var statusItem *application.MenuItem
	d := newReactDesktop(func(snapshot reactSnapshot) {
		if app != nil {
			app.Event.Emit("clipare:state", snapshot)
		}
		if tray != nil {
			tray.SetTooltip("Clipare — " + snapshot.Sync.Message)
			tray.SetIcon(reactTrayIcon(snapshot.Sync.State, snapshot.Theme))
		}
		if statusItem != nil {
			statusItem.SetLabel(snapshot.Sync.Message)
		}
	})
	app = application.New(application.Options{
		Name: "Clipare", Description: "Clipboard synchronization over NetBird",
		Assets:   application.AssetOptions{Handler: application.BundledAssetFileServer(os.DirFS(assets)), DisableLogging: true},
		Services: []application.Service{application.NewService(&ReactService{d: d})},
		Windows:  application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		Mac:      application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
	})
	d.openExternal = app.Browser.OpenURL
	refreshAppearance := func() {
		d.mu.Lock()
		if app.Env.IsDarkMode() {
			d.snapshot.Theme = "dark"
		} else {
			d.snapshot.Theme = "light"
		}
		d.snapshot.Accent = reactSystemAccent()
		d.emitLocked()
		d.mu.Unlock()
	}
	refreshAppearance()
	if runtime.GOOS == "darwin" {
		app.Event.OnApplicationEvent(events.Mac.ApplicationDidChangeTheme, func(*application.ApplicationEvent) { refreshAppearance() })
	} else {
		app.Event.OnApplicationEvent(events.Windows.SystemThemeChanged, func(*application.ApplicationEvent) { refreshAppearance() })
	}
	windowHeight := 760
	if runtime.GOOS == "darwin" {
		windowHeight = 620 // Keep the compact utility content in view without a large empty tail.
	}
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "Clipare", Title: "Clipare", Width: 700, Height: windowHeight, MinWidth: 520, MinHeight: 520,
		BackgroundColour: application.NewRGB(36, 37, 41),
	})
	var activationCleanup func()
	window.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		// The platform environment is not initialized before app.Run(). Read the
		// actual appearance when the WebView first becomes visible (and on reopen).
		refreshAppearance()
		if activationCleanup != nil {
			return
		}
		cleanup, err := installReactActivation(window, path)
		if err != nil {
			log.Warn("GUI activation unavailable", "error", err)
			return
		}
		activationCleanup = cleanup
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { window.Hide(); e.Cancel() })
	tray = app.SystemTray.New()
	tray.SetIcon(reactTrayIcon("degraded", "light"))
	tray.SetTooltip("Clipare — запуск")
	menu := app.NewMenu()
	statusItem = menu.Add("Запуск Clipare…")
	statusItem.SetEnabled(false)
	menu.AddSeparator()
	menu.Add("Открыть Clipare").OnClick(func(*application.Context) { window.Show(); window.Focus() })
	menu.Add("Приостановить / возобновить").OnClick(func(*application.Context) { _ = (&ReactService{d: d}).Perform(reactCommand{Type: "sync.toggle"}) })
	menu.Add("Проверить обновления").OnClick(func(*application.Context) {
		window.Show()
		window.Focus()
		_ = (&ReactService{d: d}).Perform(reactCommand{Type: "update.check"})
	})
	menu.AddSeparator()
	menu.Add("Выйти").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)
	tray.OnClick(func() { window.Show(); window.Focus() })
	if runtime.GOOS == "darwin" {
		app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) { window.Show(); window.Focus() })
	}
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- runDesktopReady(ctx, path, log, d, backend, nil, loopTiming{30 * time.Millisecond, 2 * time.Second, 5 * time.Second}, ready, restored)
		app.Quit()
	}()
	go func() { <-ctx.Done(); app.Quit() }()
	guiErr := app.Run()
	if activationCleanup != nil {
		activationCleanup()
	}
	cancel()
	loopErr := <-loopDone
	if guiErr != nil {
		return guiErr
	}
	return loopErr
}

func reactAssetDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	base := filepath.Dir(exe)
	candidates := []string{filepath.Join(base, "ui"), filepath.Join(base, "..", "Resources", "ui"), filepath.Join("frontend", "dist")}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", errors.New("React UI assets not found; run npm run build in frontend")
}

// HasPackagedReactUI enables the migration only for installations that ship
// the complete frontend; direct Go builds retain the legacy recovery path.
func HasPackagedReactUI() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	base := filepath.Dir(exe)
	for _, candidate := range []string{filepath.Join(base, "ui"), filepath.Join(base, "..", "Resources", "ui")} {
		if info, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

var trayIcons sync.Map

func reactTrayIcon(state, theme string) []byte {
	key := state + ":" + theme
	if cached, ok := trayIcons.Load(key); ok {
		return cached.([]byte)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	// Two opposing clipboard arrows are intentionally simple at tray size.
	ink := imgcolor.NRGBA{R: 44, G: 48, B: 54, A: 255}
	if theme == "dark" {
		ink = imgcolor.NRGBA{R: 225, G: 230, B: 235, A: 255}
	}
	for x := 8; x <= 23; x++ {
		img.Set(x, 9, ink)
		img.Set(x, 21, ink)
	}
	for i := 0; i < 6; i++ {
		img.Set(22-i, 6+i, ink)
		img.Set(9+i, 18+i, ink)
	}
	badge := imgcolor.NRGBA{R: 231, G: 153, B: 53, A: 255}
	if state == "active" {
		badge = imgcolor.NRGBA{R: 62, G: 179, B: 112, A: 255}
	}
	if state == "disabled" {
		badge = imgcolor.NRGBA{R: 215, G: 75, B: 78, A: 255}
	}
	for y := 22; y < 32; y++ {
		for x := 22; x < 32; x++ {
			dx, dy := x-27, y-27
			if dx*dx+dy*dy <= 23 {
				img.Set(x, y, badge)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(fmt.Sprintf("encode tray icon: %v", err))
	}
	result := buf.Bytes()
	trayIcons.Store(key, result)
	return result
}
