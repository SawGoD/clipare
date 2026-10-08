//go:build windows

package ui

import (
	"clipare/internal/config"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
)

// Opt-in synthetic native UI: no clipboard, config, NetBird or autostart access.
func TestWindowsFluentPreview(t *testing.T) {
	dir := os.Getenv("CLIPARE_UI_SCREENSHOTS_DIR")
	if dir == "" {
		t.Skip("native Windows preview is opt-in")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	n := &nativeDesktop{scale: 1}
	if err := n.Init(); err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	peers := []config.Peer{{ID: "a", Name: "MacBook"}, {ID: "b", Name: "Workstation-Development-Long-Device-Name"}}
	f := form{}
	f.Values[0] = "Desktop-PC"
	f.Values[1] = "synthetic-device"
	f.Values[2] = "100.64.0.1"
	f.Values[3] = "45873"
	f.Values[5] = "Протокол 1 · Тест интерфейса"
	f.Values[9] = "45873"
	n.Show(f, peers, []string{"100.64.0.1"})
	n.Update("Синхронизация включена", true, peers, map[string]bool{"a": true})
	n.UpdateSettings("0.5.0", true)
	base := systemTheme()
	variants := []struct {
		name  string
		theme winTheme
	}{{"light", base}, {"dark", base}, {"purple", base}}
	variants[0].theme.dark = false
	variants[0].theme.background = color(0xf3f3f3)
	variants[0].theme.surface = color(0xffffff)
	variants[0].theme.text = color(0x1b1b1b)
	variants[0].theme.muted = color(0x616161)
	variants[0].theme.border = color(0xd8d8d8)
	variants[1].theme.dark = true
	variants[1].theme.background = color(0x202020)
	variants[1].theme.surface = color(0x2b2b2b)
	variants[1].theme.text = color(0xf5f5f5)
	variants[1].theme.muted = color(0xbcbcbc)
	variants[1].theme.border = color(0x494949)
	variants[2].theme = variants[0].theme
	variants[2].theme.accent = color(0x744da9)
	variants[2].theme.onAccent = color(0xffffff)
	for _, v := range variants {
		n.setTheme(v.theme)
		n.Poll()
		captureWindow(t, n.home, filepath.Join(dir, "home-"+v.name+".png"))
	}
	n.setTheme(variants[0].theme)
	n.Show(f, nil, nil)
	if call("IsWindowVisible", n.emptyDevices) == 0 {
		t.Fatal("empty device state hidden")
	}
	captureWindow(t, n.home, filepath.Join(dir, "home-empty.png"))
	n.Discovered("Desktop-PC — 100.64.0.2\nLaptop — laptop.netbird.cloud", "Выберите устройство и нажмите «Подключить»")
	if len(n.rows[n.foundList]) != 2 {
		t.Fatal("discovery rows missing")
	}
	captureWindow(t, n.found, filepath.Join(dir, "discovery.png"))
	n.Discovered("", "Автоматическое обнаружение недоступно")
	if call("IsWindowEnabled", call("GetDlgItem", n.found, 13)) != 0 {
		t.Fatal("connect enabled without devices")
	}
	captureWindow(t, n.found, filepath.Join(dir, "discovery-empty.png"))
	n.Pair("MacBook хочет подключиться к Clipare", "482 731", 1)
	if windowText(n.pairCode) != "482 731" || call("GetDlgCtrlID", n.pairReject) != eventReject {
		t.Fatal("incoming pairing controls")
	}
	captureWindow(t, n.pair, filepath.Join(dir, "pairing.png"))
	n.Pair("MacBook", "482 731", 2)
	if call("GetDlgCtrlID", n.pairAllow) != eventConfirmLocal {
		t.Fatal("local confirmation action changed")
	}
	n.Pair("MacBook", "482 731", 0)
	if call("IsWindowVisible", n.pairAllow) != 0 {
		t.Fatal("outgoing approval exposed")
	}
	n.PairClose()
	states := []struct {
		name string
		p    updatePrompt
	}{
		{"latest", updateNotice("Установлена последняя версия Clipare", "0.5.0")},
		{"available", updateOffer("0.5.0", "0.6.0")},
		{"download", updateProgress("Загрузка Clipare 0.6.0…")},
		{"ready", updateProgress("Обновление готово\n\nClipare перезапустится для установки…")},
		{"error", updateFailure("GitHub недоступен. Проверьте подключение к Интернету.", "0.5.0", eventCheckUpdate)},
	}
	for _, s := range states {
		n.UpdatePrompt(s.p)
		if windowText(n.updateDismiss) != s.p.Dismiss {
			t.Fatal("dismiss label does not match state")
		}
		if call("GetDlgCtrlID", n.updateInstall) != uintptr(s.p.Action) {
			t.Fatal("wrong update action")
		}
		captureWindow(t, n.updateWindow, filepath.Join(dir, "update-"+s.name+".png"))
	}
	call("ShowWindow", n.window, 5)
	captureWindow(t, n.window, filepath.Join(dir, "advanced.png"))
}

func captureWindow(t *testing.T, hwnd uintptr, path string) {
	t.Helper()
	call("UpdateWindow", hwnd)
	var r winRect
	call("GetWindowRect", hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if w <= 0 || h <= 0 || w*h > 16000000 {
		t.Fatal("invalid screenshot size")
	}
	dc := call("GetWindowDC", hwnd)
	defer call("ReleaseDC", hwnd, dc)
	mem := gcall("CreateCompatibleDC", dc)
	defer gcall("DeleteDC", mem)
	bmp := gcall("CreateCompatibleBitmap", dc, uintptr(w), uintptr(h))
	defer gcall("DeleteObject", bmp)
	old := gcall("SelectObject", mem, bmp)
	if call("PrintWindow", hwnd, mem, 2) == 0 {
		gcall("SelectObject", mem, old)
		t.Fatal("PrintWindow failed")
	}
	gcall("SelectObject", mem, old)
	var header struct {
		Size                   uint32
		Width, Height          int32
		Planes, Bits           uint16
		Compression, ImageSize uint32
		XPels, YPels           int32
		Used, Important        uint32
	}
	header.Size = 40
	header.Width = int32(w)
	header.Height = -int32(h)
	header.Planes = 1
	header.Bits = 32
	pixels := make([]byte, w*h*4)
	if gcall("GetDIBits", mem, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), 0) == 0 {
		t.Fatal("GetDIBits failed")
	}
	for i := 0; i < len(pixels); i += 4 {
		pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
		pixels[i+3] = 255
	}
	img := &image.RGBA{Pix: pixels, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, img)
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
