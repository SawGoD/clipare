//go:build windows

package ui

import (
	"clipare/internal/config"
	"clipare/internal/instance"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
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
	profile := filepath.Join(t.TempDir(), "config.yaml")
	guard, err := instance.Acquire(context.Background(), profile)
	if err != nil || !guard.Primary {
		t.Fatal("fixture did not acquire primary GUI", err)
	}
	defer guard.Close()
	n.Instance(profile)
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
	// Production can receive pairing before the first settings/open action.
	n.Prepare(f, peers, []string{"100.64.0.1"})
	if call("IsWindowVisible", n.main) != 0 || n.Read().Values != f.Values {
		t.Fatal("tray startup must bind fields without opening the window")
	}
	n.Pair("MacBook", "482 731", 1)
	if windowText(n.homeName) != f.Values[0] {
		t.Fatal("incoming pairing opened uninitialized fields")
	}
	n.PairClose()
	n.Show(f, peers, []string{"100.64.0.1"})
	call("ShowWindow", n.main, 6)
	if call("IsIconic", n.main) == 0 {
		t.Fatal("fixture did not minimize the window")
	}
	visible(n.main, false)
	n.Show(f, peers, []string{"100.64.0.1"})
	if call("IsIconic", n.main) != 0 || call("IsWindowVisible", n.main) == 0 {
		t.Fatal("opening from tray did not restore the minimized hidden window")
	}
	if n.controls[n.advancedButton].bounds.y != n.controls[n.pauseButton].bounds.y || n.icons[n.advancedButton] != "\ue713" {
		t.Fatal("advanced gear is not beside pause")
	}
	n.events = nil
	for _, notification := range []uintptr{0x202, 0x400 | (1 << 16), 0x401 | (1 << 16)} {
		visible(n.main, false)
		call("PostMessageW", n.main, 0x8001, 0, notification)
		n.Poll()
		if call("IsWindowVisible", n.main) == 0 || windowText(n.homeName) != f.Values[0] {
			t.Fatal("queued tray activation failed or reset device name", notification)
		}
	}
	// Repeat launch must activate this hidden GUI, without another listener.
	visible(n.main, false)
	activation := make(chan error, 1)
	go func() {
		second, e := instance.Acquire(context.Background(), profile)
		if e == nil {
			if second.Primary {
				e = fmt.Errorf("second instance became primary")
			}
			second.Close()
		}
		activation <- e
	}()
	deadline := time.Now().Add(6 * time.Second)
	activated := false
	for time.Now().Before(deadline) {
		n.Poll()
		select {
		case e := <-activation:
			if e != nil {
				t.Fatal(e)
			}
			activated = true
		default:
		}
		if activated {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !activated || call("IsWindowVisible", n.main) == 0 {
		t.Fatal("repeat launch did not activate the existing window")
	}
	// Continuous messages must not starve an already queued application action.
	n.events = append(n.events, eventSettings)
	for i := 0; i < 128; i++ {
		call("PostMessageW", n.main, 0, 0, 0)
	}
	if n.Poll() != eventSettings {
		t.Fatal("message burst starved tray action")
	}
	n.ApplyVisibility(applyMask(f, f, false))
	nameApply := call("GetDlgItem", n.home, eventDeviceName)
	if call("IsWindowVisible", nameApply) != 0 {
		t.Fatal("unchanged name shows apply")
	}
	n.set(0, "Changed name")
	n.ApplyVisibility(applyMask(n.Read(), f, false))
	if call("IsWindowVisible", nameApply) == 0 {
		t.Fatal("changed name hides apply")
	}
	n.set(0, f.Values[0])
	n.ApplyVisibility(applyMask(n.Read(), f, false))
	if call("IsWindowVisible", nameApply) != 0 {
		t.Fatal("reverted name shows apply")
	}
	n.navigation.AdvancedExpanded = true
	n.layoutPreferences()
	advancedApply := call("GetDlgItem", n.home, eventSave)
	if call("IsWindowVisible", advancedApply) != 0 {
		t.Fatal("advanced expansion shows unchanged apply")
	}
	n.set(3, "45874")
	n.ApplyVisibility(applyMask(n.Read(), f, false))
	if call("IsWindowVisible", advancedApply) == 0 {
		t.Fatal("changed port hides apply")
	}
	n.set(3, f.Values[3])
	n.ApplyVisibility(applyMask(n.Read(), f, false))
	if call("IsWindowVisible", advancedApply) != 0 {
		t.Fatal("reverted port shows apply")
	}
	n.navigation.AdvancedExpanded = false
	n.layoutPreferences()
	if len(n.rowTrash) != len(peers) {
		t.Fatal("missing per-device removal buttons")
	}
	// Test hit testing, not just existence: the list must not cover trash.
	n.Update("Синхронизация включена", true, peers, map[string]bool{"a": true})
	call("UpdateWindow", n.home)
	var trashRect winRect
	trash := n.rowTrash[0]
	call("GetWindowRect", trash, uintptr(unsafe.Pointer(&trashRect)))
	x, y := (trashRect.Left+trashRect.Right)/2, (trashRect.Top+trashRect.Bottom)/2
	point := uintptr(uint32(x)) | uintptr(uint32(y))<<32
	if call("WindowFromPoint", point) != trash {
		t.Fatal("device list covers the removal button")
	}
	// Answer the real modal confirmation while its nested message loop runs.
	answered := make(chan bool, 1)
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			dialog := call("FindWindowW", uintptr(unsafe.Pointer(wide("#32770"))), uintptr(unsafe.Pointer(wide("Удалить устройство?"))))
			if dialog != 0 && call("GetWindow", dialog, 4) == n.main {
				call("PostMessageW", dialog, 0x111, 6, 0)
				answered <- true
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		answered <- false
	}()
	n.events = nil
	call("SendMessageW", trash, 0x201, 1, 8|(8<<16))
	call("SendMessageW", trash, 0x202, 0, 8|(8<<16))
	if !<-answered || n.Selected() != 0 || len(n.events) == 0 || n.events[len(n.events)-1] != eventRemove {
		t.Fatal("mouse click did not confirm removal of the correct device")
	}
	n.events = nil
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
	n.Platforms(map[string]string{"a": "darwin", "b": "windows"})
	n.Update("Синхронизация включена", true, peers, map[string]bool{"a": true})
	if n.rows[n.homeList][0].platform != "darwin" || n.rows[n.homeList][1].platform != "windows" {
		t.Fatal("platform refresh omitted")
	}
	n.UpdateSettings("0.5.0", true)
	captureWindow(t, n.home, filepath.Join(dir, "home-painted.png"))
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
	if n.activePanel() != n.home || call("IsWindowVisible", n.homeName) == 0 || call("IsWindowVisible", n.homeStatus) == 0 || call("IsWindowVisible", n.homeList) != 0 {
		t.Fatal("discovery did not stay in the home device card")
	}
	n.Update("Синхронизация включена", true, peers, nil)
	if call("IsWindowVisible", n.homeList) != 0 {
		t.Fatal("peer health refresh replaced inline discovery")
	}
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
	n.back(false)
	if n.navigation.View != ViewHome || call("IsWindowVisible", n.homeList) == 0 || call("IsWindowVisible", n.foundStatus) != 0 {
		t.Fatal("back did not restore the peer card")
	}
	n.Discovered("Desktop-PC — 100.64.0.2", "Выберите устройство")
	captureWindow(t, n.home, filepath.Join(dir, "discovery-one.png"))
	if n.controls[n.foundList].bounds.h != 60 {
		t.Fatal("single discovery row left a large blank list")
	}
	n.Pair("MacBook хочет подключиться к Clipare", "482 731", 1)
	if n.activePanel() != n.home || call("IsWindowVisible", n.homeName) == 0 || call("IsWindowVisible", n.homeList) != 0 || call("IsWindowVisible", n.foundList) != 0 {
		t.Fatal("pairing replaced home or overlaps device list")
	}
	n.Update("Синхронизация включена", true, peers, map[string]bool{"a": true})
	if call("IsWindowVisible", n.homeList) != 0 {
		t.Fatal("peer refresh overlaps pairing")
	}
	if n.navigation.View != ViewPairing {
		t.Fatal("pairing did not navigate inline")
	}
	if windowText(n.pairCode) != "482 731" || call("GetDlgCtrlID", n.pairReject) != eventReject {
		t.Fatal("incoming pairing controls")
	}
	captureWindow(t, n.pair, filepath.Join(dir, "pairing.png"))
	card := n.scaledRect(n.home, n.windows[n.home].cards[1])
	for _, h := range []uintptr{n.pairReject, n.pairAllow} {
		var rect winRect
		call("GetWindowRect", h, uintptr(unsafe.Pointer(&rect)))
		call("MapWindowPoints", 0, n.home, uintptr(unsafe.Pointer(&rect)), 2)
		if rect.Top < card.Top || rect.Bottom > card.Bottom {
			t.Fatal("pairing action escaped device card")
		}
	}
	n.Pair("MacBook", "482 731", 2)
	if call("GetDlgCtrlID", n.pairAllow) != eventConfirmLocal {
		t.Fatal("local confirmation action changed")
	}
	n.Pair("MacBook", "482 731", 0)
	if call("IsWindowVisible", n.pairAllow) != 0 {
		t.Fatal("outgoing approval exposed")
	}
	n.PairClose()
	if n.activePanel() != n.home || call("IsWindowVisible", n.pairCode) != 0 {
		t.Fatal("completed pairing did not restore home")
	}
	for _, mode := range []int{0, 1, 2} {
		n.Pair("MacBook", "482 731", mode)
		n.events = nil
		n.back(false)
		want := eventCancelPair
		if mode == 1 {
			want = eventReject
		}
		if len(n.events) != 1 || n.events[0] != want || n.navigation.View != ViewHome {
			t.Fatal("inline back lost pairing cancellation")
		}
	}
	if len(n.helpTexts) != 4 {
		t.Fatal("advanced help missing")
	}
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
	if filepath.Base(path) == "home-painted.png" {
		call("RedrawWindow", hwnd, 0, 0, 0x185)
		if gcall("BitBlt", mem, 0, 0, uintptr(w), uintptr(h), dc, 0, 0, 0x00cc0020) == 0 {
			t.Fatal("capture of actual painted controls failed")
		}
	} else if call("PrintWindow", hwnd, mem, 2) == 0 {
		gcall("SelectObject", mem, old)
		t.Fatal("PrintWindow failed")
	}
	// DefWindowProc's WM_PRINT path explicitly asks for background + children;
	// it avoids a DWM first-frame black client area in headless runners.
	if filepath.Base(path) != "home-painted.png" {
		call("SendMessageW", hwnd, 0x317, mem, 0x1e)
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
