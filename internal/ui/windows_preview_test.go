//go:build windows

package ui

import (
	"clipare/internal/config"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
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
	for h := range n.windows {
		if h != n.main && call("GetWindowLongPtrW", h, ^uintptr(15))&0x40000000 == 0 {
			t.Fatal("workflow panel is still a top-level window")
		}
	}
	peers := []config.Peer{{ID: "a", Name: "MacBook"}, {ID: "b", Name: "Workstation-Development-Long-Device-Name"}}
	f := form{}
	f.Values[0] = "Desktop-PC"
	f.Values[1] = "synthetic-device"
	f.Values[2] = "100.64.0.1"
	f.Values[3] = "45873"
	f.Values[5] = "Протокол 1 · Тест интерфейса"
	f.Values[9] = "45873"
	n.Show(f, peers, []string{"100.64.0.1"})
	if len(n.rowTrash) != len(peers) {
		t.Fatal("missing per-device removal buttons")
	}
	if call("IsWindowEnabled", call("GetDlgItem", n.home, eventImport)) != 0 {
		t.Fatal("empty connection code enabled import")
	}
	n.set(10, "clipare1:synthetic")
	n.Actions(actionMask(n.Read()))
	if call("IsWindowEnabled", call("GetDlgItem", n.home, eventImport)) == 0 {
		t.Fatal("nonempty connection code disabled import")
	}
	n.set(10, "")
	n.Actions(actionMask(n.Read()))
	n.Update("Синхронизация включена", true, peers, map[string]bool{"a": true})
	n.UpdateSettings("0.5.0", true)
	for _, state := range []SyncState{SyncActive, SyncDisabled, SyncDegraded, SyncActive} {
		n.SyncStatus(SyncStatus{State: state, Message: "Тест статуса", Reason: "Синтетический preview"})
		if n.statusIcons[state] == 0 {
			t.Fatal("status icon creation failed")
		}
	}
	if call("IsWindowVisible", n.homeAuto) != 0 || call("IsWindowVisible", n.updateCheck) != 0 {
		t.Fatal("preferences must be collapsed initially")
	}
	n.preferencesExpanded = true
	n.layoutPreferences()
	if call("IsWindowVisible", n.homeAuto) == 0 || call("IsWindowVisible", n.updateCheck) == 0 {
		t.Fatal("expanded preferences are hidden")
	}
	captureWindow(t, n.home, filepath.Join(dir, "home-expanded.png"))
	n.preferencesExpanded = false
	n.layoutPreferences()
	// Painting must not replace the native automatic-checkbox state machine.
	before := call("SendMessageW", n.homeAuto, 0xf0, 0, 0)
	call("SendMessageW", n.homeAuto, 0xf5, 0, 0) // BM_CLICK
	if call("SendMessageW", n.homeAuto, 0xf0, 0, 0) == before {
		t.Fatal("styled checkbox no longer toggles on native click")
	}
	call("SendMessageW", n.homeAuto, 0xf1, before, 0)
	call("SendMessageW", n.homeName, 0xc, 0, uintptr(unsafe.Pointer(wide("Desktop-PC"))))
	if windowText(n.homeName) != "Desktop-PC" {
		t.Fatal("styled edit no longer accepts native text updates")
	}
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
	variants[1].theme.danger = color(0xffa7ab)
	variants[1].theme.online = color(0x8abf9b)
	variants[2].theme = variants[0].theme
	variants[2].theme.accent = color(0x744da9)
	variants[2].theme.onAccent = color(0xffffff)
	for _, v := range variants {
		n.setTheme(v.theme)
		n.Poll()
		captureWindow(t, n.home, filepath.Join(dir, "home-"+v.name+".png"))
	}
	n.scrollWindow(n.home, 1, 10000, true)
	captureWindow(t, n.home, filepath.Join(dir, "home-scrolled.png"))
	n.scrollWindow(n.home, 1, 0, true)
	n.setTheme(variants[0].theme)
	for key, font := range n.fonts {
		dc := gcall("CreateCompatibleDC", 0)
		old := gcall("SelectObject", dc, font)
		var face [128]uint16
		gcall("GetTextFaceW", dc, 128, uintptr(unsafe.Pointer(&face[0])))
		gcall("SelectObject", dc, old)
		gcall("DeleteDC", dc)
		t.Logf("font dpi=%d role=%d: %s", key[0], key[1], syscall.UTF16ToString(face[:]))
	}
	n.Show(f, nil, nil)
	if call("IsWindowVisible", n.emptyDevices) == 0 {
		t.Fatal("empty device state hidden")
	}
	if call("IsWindowVisible", n.emptyAdd) == 0 || call("IsWindowVisible", n.removePeer) != 0 || call("IsWindowVisible", n.compactAdd) != 0 || call("IsWindowVisible", n.homeList) != 0 {
		t.Fatal("empty devices expose list actions")
	}
	captureWindow(t, n.home, filepath.Join(dir, "home-empty.png"))
	n.Update("Синхронизация включена", true, peers, nil)
	if call("IsWindowVisible", n.emptyAdd) != 0 || len(n.rowTrash) != len(peers) || call("IsWindowVisible", n.rowTrash[0]) == 0 {
		t.Fatal("peer arrival did not restore list actions")
	}
	n.Discovered("Desktop-PC — 100.64.0.2\nLaptop — laptop.netbird.cloud", "Выберите устройство и нажмите «Подключить»")
	if call("GetWindowLongPtrW", n.found, ^uintptr(15))&0x00100000 != 0 {
		t.Fatal("switching from a tall home view left a horizontal scrollbar")
	}
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
	if n.navigation.View != ViewPairing {
		t.Fatal("pairing did not navigate inline")
	}
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
	n.navigate(ViewHome)
	n.navigation.AdvancedExpanded = true
	n.layoutPreferences()
	n.scrollWindow(n.home, 1, int32(n.controls[n.fields[1]].bounds.y-40), true)
	captureWindow(t, n.window, filepath.Join(dir, "advanced.png"))
	call("SendMessageW", n.fields[2], 0x143, 0, uintptr(unsafe.Pointer(wide("100.64.0.2"))))
	var combo comboInfo
	combo.Size = uint32(unsafe.Sizeof(combo))
	call("GetComboBoxInfo", n.fields[2], uintptr(unsafe.Pointer(&combo)))
	for _, v := range variants[:2] {
		n.setTheme(v.theme)
		call("SendMessageW", n.fields[2], 0x14f, 1, 0)
		n.Poll()
		if !n.comboOpen() {
			t.Fatal("native dropdown did not open")
		}
		captureWindow(t, n.home, filepath.Join(dir, "advanced-"+v.name+".png"))
		captureWindow(t, combo.List, filepath.Join(dir, "netbird-dropdown-"+v.name+".png"))
		call("SendMessageW", n.fields[2], 0x14f, 0, 0)
	}
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
	// DefWindowProc's WM_PRINT path explicitly asks for background + children;
	// it avoids a DWM first-frame black client area in headless runners.
	call("SendMessageW", hwnd, 0x317, mem, 0x1e)
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
