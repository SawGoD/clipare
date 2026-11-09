//go:build windows || (darwin && cgo)

package ui

import (
	"bytes"
	"clipare/internal/clipboard"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"image"
	imgcolor "image/color"
	"image/draw"
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
	var window *application.WebviewWindow
	wasIncomingPair := false
	d := newReactDesktop(func(snapshot reactSnapshot) {
		incomingPair := snapshot.Pairing != nil && snapshot.Pairing.Direction == "incoming"
		if incomingPair && !wasIncomingPair && window != nil {
			// Publishing happens under the desktop state lock. Showing the window
			// asynchronously avoids re-entering that lock via WindowShow.
			go func() {
				if !window.IsVisible() {
					window.Show()
					window.Focus()
				}
			}()
		}
		wasIncomingPair = incomingPair
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
		Name: "Clipare", Description: "Clipboard synchronization over NetBird", Icon: appIcon,
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
	minWindowHeight := 520
	if runtime.GOOS == "darwin" {
		windowHeight = 520 // Content fits without the unused tail; expanded settings scroll.
		minWindowHeight = 480
	}
	windowOptions := application.WebviewWindowOptions{
		Name: "Clipare", Title: "Clipare", Width: 700, Height: windowHeight, MinWidth: 520, MinHeight: minWindowHeight,
		BackgroundColour: application.NewRGB(36, 37, 41),
	}
	if runtime.GOOS == "windows" && reactWindowsBackdropSupported() {
		windowOptions.BackgroundType = application.BackgroundTypeTranslucent
		windowOptions.Windows.BackdropType = application.Acrylic
		d.snapshot.WindowsBackdrop = true
	}
	if runtime.GOOS == "darwin" {
		// Native NSGlassEffectView on macOS 26; Wails falls back to
		// NSVisualEffectView on older supported versions.
		windowOptions.Mac.Backdrop = application.MacBackdropLiquidGlass
		windowOptions.Mac.LiquidGlass.Material = application.NSVisualEffectMaterialWindowBackground
		windowOptions.BackgroundColour = application.NewRGBA(0, 0, 0, 0)
	}
	window = app.Window.NewWithOptions(windowOptions)
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

//go:embed icon/clipare.png
var appIcon []byte

//go:embed icon/tray.png
var trayIconPNG []byte

var trayIcons sync.Map

func reactTrayIcon(state, theme string) []byte {
	key := state + ":" + theme
	if cached, ok := trayIcons.Load(key); ok {
		return cached.([]byte)
	}
	base, err := png.Decode(bytes.NewReader(trayIconPNG))
	if err != nil {
		panic("invalid embedded tray icon")
	}
	img := image.NewNRGBA(base.Bounds())
	draw.Draw(img, img.Bounds(), base, base.Bounds().Min, draw.Src)
	badge := imgcolor.NRGBA{R: 231, G: 153, B: 53, A: 255}
	if state == "active" {
		badge = imgcolor.NRGBA{R: 48, G: 164, B: 96, A: 255}
	}
	if state == "disabled" {
		badge = imgcolor.NRGBA{R: 207, G: 61, B: 67, A: 255}
	}
	for y := 39; y < 64; y++ {
		for x := 39; x < 64; x++ {
			dx, dy := x-51, y-51
			d := dx*dx + dy*dy
			if d <= 144 {
				img.Set(x, y, imgcolor.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
			if d <= 81 {
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
