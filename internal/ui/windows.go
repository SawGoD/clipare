//go:build windows

package ui

import (
	"clipare/internal/config"
	"errors"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var win = syscall.NewLazyDLL("user32.dll")
var shell = syscall.NewLazyDLL("shell32.dll")
var k32 = syscall.NewLazyDLL("kernel32.dll")

type windowClass struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	SmallIcon                          uintptr
}
type message struct {
	Window         uintptr
	ID             uint32
	WParam, LParam uintptr
	Time           uint32
	X, Y           int32
	Private        uint32
}
type notifyIcon struct {
	Size                uint32
	Window              uintptr
	ID, Flags, Callback uint32
	Icon                uintptr
	Tip                 [128]uint16
	State, StateMask    uint32
	Info                [256]uint16
	Timeout             uint32
	InfoTitle           [64]uint16
	InfoFlags           uint32
	GUID                [16]byte
	BalloonIcon         uintptr
}
type nativeDesktop struct {
	updateWindow, updateCheck, updateVersion, updateText, updateInstall, updateDismiss uintptr
	window, instance, callback, icon                                                   uintptr
	fields                                                                             [11]uintptr
	auto, list                                                                         uintptr
	statusLabel                                                                        uintptr
	events                                                                             []int
	state                                                                              string
	enabled                                                                            bool
	peerLines                                                                          string
	scale                                                                              float64
	taskbar                                                                            uint32
	home, found, pair                                                                  uintptr
	homeName, homeAuto, homeList, homeStatus                                           uintptr
	foundList, foundStatus, pairName, pairCode, pairHelp, pairAllow, pairReject        uintptr
	pairIncoming                                                                       bool
	theme                                                                              winTheme
	surfaceBrush, backgroundBrush                                                      uintptr
	windows                                                                            map[uintptr]winWindow
	controls                                                                           map[uintptr]winControl
	fonts                                                                              map[[2]int]uintptr
	rows                                                                               map[uintptr][]deviceRow
	rowTrash                                                                           []uintptr
	rowCallback                                                                        uintptr
	footerCheck                                                                        uintptr
	devicesHeader                                                                      uintptr
	discoveryControls                                                                  []uintptr
	pairControls                                                                       []uintptr
	pairMode                                                                           int
	pauseButton, emptyDevices, emptyDiscovery                                          uintptr
	layingOut                                                                          bool
	lastFocus                                                                          uintptr
	controlCallback                                                                    uintptr
	emptyAdd, compactAdd, removePeer, preferencesToggle, advancedButton                uintptr
	preferencesExpanded                                                                bool
	main, notice, noticeText                                                           uintptr
	navigation                                                                         Navigation
	advancedControls                                                                   []uintptr
	advancedBase                                                                       map[uintptr]int
	utilityStatus                                                                      SyncStatus
	statusDot                                                                          uintptr
	icons                                                                              map[uintptr]string
	tooltip                                                                            uintptr
	tooltipText                                                                        []*uint16
	statusIcons                                                                        [3]uintptr
	actionButtons                                                                      map[uintptr]int
	applyVisible                                                                       uint64
	helpTexts                                                                          map[int]string
}

func newDesktop() (desktop, error) { return &nativeDesktop{scale: 1}, nil }
func wide(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(s, "\x00", ""))
	return p
}
func call(name string, args ...uintptr) uintptr { r, _, _ := win.NewProc(name).Call(args...); return r }
func (n *nativeDesktop) px(v int) uintptr       { return uintptr(int(float64(v) * n.scale)) }
func (n *nativeDesktop) control(class, text string, style uintptr, x, y, w, h, id int) uintptr {
	kind := 0
	if class == "BUTTON" && style&15 == 0 {
		style = style&^15 | 11
		switch id {
		case 1, 8, 13, 14, 18, 20, eventDeviceName:
			kind = 1
		case 9:
			kind = 2
		}
	}
	r := n.scaledRect(n.window, logicalRect{x, y, w, h})
	handle := call("CreateWindowExW", 0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(text))), style|0x50000000, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), n.window, uintptr(id), n.instance, 0)
	n.controls[handle] = winControl{parent: n.window, class: class, bounds: logicalRect{x, y, w, h}, kind: kind}
	if class == "BUTTON" && (id == eventSave || id == eventCopy || id == eventImport || id == eventUpsert || id == eventDeviceName) {
		if n.actionButtons == nil {
			n.actionButtons = map[uintptr]int{}
		}
		n.actionButtons[handle] = id
		if id == eventSave || id == eventDeviceName {
			visible(handle, false)
		}
	}
	call("SendMessageW", handle, 0x30, n.fontFor(n.window, 0), 1)
	if class == "EDIT" || class == "COMBOBOX" || class == "BUTTON" && style&15 == 3 {
		n.styleControl(handle)
	}
	return handle
}
func (n *nativeDesktop) label(text string, x, y, w int) {
	n.control("STATIC", text, 0x4000, x, y, w, 22, 0)
}
func (n *nativeDesktop) input(i, x, y, w int, secure bool) {
	style := uintptr(0x00810080)
	if secure {
		style |= 0x20
	}
	n.fields[i] = n.control("EDIT", "", style, x, y, w, 26, 100+i)
}
func (n *nativeDesktop) button(text string, x, y, w, tag int) {
	n.control("BUTTON", text, 0x10000, x, y, w, 30, tag)
}
func (n *nativeDesktop) Init() error {
	common := struct{ Size, Classes uint32 }{8, 0x40ff}
	comctl.NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&common)))
	call("SetProcessDpiAwarenessContext", ^uintptr(3)) // per-monitor v2
	if dpi := call("GetDpiForSystem"); dpi > 0 {
		n.scale = float64(dpi) / 96
	}
	n.instance, _, _ = k32.NewProc("GetModuleHandleW").Call(0)
	n.windows = make(map[uintptr]winWindow)
	n.controls = make(map[uintptr]winControl)
	n.fonts = make(map[[2]int]uintptr)
	n.rows = make(map[uintptr][]deviceRow)
	n.icons = make(map[uintptr]string)
	n.applyTheme()
	n.icon = statusIcon(SyncStatus{State: SyncDegraded})
	if n.icon == 0 {
		n.icon = call("LoadIconW", 0, 32512)
	} else {
		n.statusIcons[SyncDegraded] = n.icon
	}
	n.taskbar = uint32(call("RegisterWindowMessageW", uintptr(unsafe.Pointer(wide("TaskbarCreated")))))
	n.callback = syscall.NewCallback(func(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
		if msg == n.taskbar && n.taskbar != 0 {
			n.addTray()
			return 0
		}
		if r, ok := n.themeMessage(hwnd, msg, w, l); ok {
			return r
		}
		if n.scrollMessage(hwnd, msg, w, l) {
			return 0
		}
		switch msg {
		case 0x2e0:
			n.dpiChanged(hwnd, w, l)
			return 0
		case 0x10:
			n.back(hwnd == n.main)
			return 0
		case 0x16:
			if w != 0 {
				n.events = append(n.events, eventQuit)
			}
			return 0
		case 0x111:
			id := int(w & 0xffff)
			if help, ok := n.helpTexts[id]; ok {
				n.Alert(help)
				return 0
			}
			if id >= 9000 && id < 9000+len(n.rowTrash) {
				n.confirmRowRemoval(id - 9000)
				return 0
			}
			if (w>>16 == 0x300 && (id >= 100 && id <= 110 || id == 130)) || (id == 102 && (w>>16 == 5 || w>>16 == 1)) {
				n.events = append(n.events, eventFormChanged)
				return 0
			}
			if id == 32 {
				n.preferencesExpanded = !n.preferencesExpanded
				n.layoutPreferences()
				return 0
			}
			if id == 30 {
				if n.navigation.View == ViewDiscovery {
					n.events = append(n.events, eventCloseDiscovery)
				}
				if n.navigation.View != ViewHome {
					n.navigate(ViewHome)
					n.navigation.AdvancedExpanded = true
				} else {
					n.navigation.AdvancedExpanded = !n.navigation.AdvancedExpanded
				}
				n.layoutPreferences()
				return 0
			}
			if id == 31 {
				n.back(false)
				return 0
			}
			if id == 1 && hwnd == n.home {
				n.set(0, windowText(n.homeName))
				call("SendMessageW", n.auto, 0xF1, call("SendMessageW", n.homeAuto, 0xF0, 0, 0), 0)
			}
			if id == 120 && w>>16 == 1 {
				n.events = append(n.events, eventSelect)
			} else if id >= 1 && id <= eventDeviceName {
				n.events = append(n.events, id)
			}
			return 0
		case 0x8001:
			if l == 0x405 {
				n.events = append(n.events, eventSettings)
				return 0
			}
			if l == 0x205 || l == 0x202 {
				n.menu()
			} else if l == 0x203 {
				n.events = append(n.events, eventSettings)
			}
			return 0
		}
		return call("DefWindowProcW", hwnd, uintptr(msg), w, l)
	})
	class := wide("ClipareSettingsWindow")
	wc := windowClass{Proc: n.callback, Instance: n.instance, Icon: n.icon, Cursor: call("LoadCursorW", 0, 32512), Name: class}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if call("RegisterClassExW", uintptr(unsafe.Pointer(&wc))) == 0 {
		return errors.New("Не удалось зарегистрировать окно Clipare")
	}
	n.main = n.panel(class, "Clipare", 568, 688)
	if n.main == 0 {
		return errors.New("Не удалось создать окно Clipare")
	}
	n.state = "Настройте подключение"
	n.installHome(class)
	n.installAdvanced()
	n.layoutPreferences()
	n.showDevicePresentation(0)
	n.applyTheme()
	n.addTray()
	return nil
}
func windowText(h uintptr) string {
	size := call("GetWindowTextLengthW", h)
	if size > 8192 {
		size = 8192
	}
	b := make([]uint16, size+1)
	call("GetWindowTextW", h, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return syscall.UTF16ToString(b)
}
func (n *nativeDesktop) installHome(class *uint16) {
	n.home = n.panel(class, "Clipare", 568, 624, logicalRect{24, 64, 520, 128}, logicalRect{24, 208, 520, 228}, logicalRect{24, 452, 520, 48})
	n.window = n.home
	n.heading("Clipare", 24, 16, 300, 1)
	n.heading("Этот компьютер", 40, 80, 280, 2)
	n.homeName = n.control("EDIT", "", 0x00810080, 40, 116, 332, 30, 130)
	n.button("Применить", 384, 116, 144, eventDeviceName)
	n.homeStatus = n.control("STATIC", "Ожидание NetBird", 0x4000, 40, 158, 312, 22, 0)
	n.statusDot = n.control("STATIC", "", 13, 24, 158, 12, 22, 0)
	n.pauseButton = n.control("BUTTON", "Пауза", 0x10000, 384, 152, 144, 32, eventPause)
	n.devicesHeader = n.heading(devicesTitle, 40, 224, 360, 2)
	n.homeList = n.deviceList(40, 264, 488, 120, 120)
	n.emptyAdd = n.control("BUTTON", addDeviceTitle, 0x10000, 244, 278, 80, 80, 11)
	c := n.controls[n.emptyAdd]
	c.kind = 3
	n.controls[n.emptyAdd] = c
	n.emptyDevices = n.control("STATIC", addDeviceTitle, 1, 56, 374, 456, 24, 0)
	n.compactAdd = n.control("BUTTON", "+ "+addDeviceTitle, 0x10000, 40, 392, 268, 30, 11)
	n.updateVersion = n.control("STATIC", "", 0x4000, 40, 576, 400, 24, 0)
	n.setRole(n.updateVersion, 4)
	n.footerCheck = n.iconButton("Проверить обновления", "\ue72c", 492, 568, 36, 19)
	n.preferencesToggle = n.control("BUTTON", additionalTitle, 0x10000, 40, 460, 488, 32, 32)
	c = n.controls[n.preferencesToggle]
	c.kind = 4
	n.controls[n.preferencesToggle] = c
	n.homeAuto = n.control("BUTTON", "Запускать при входе в систему", 0x10003, 40, 508, 488, 28, eventAutostartPreference)
	n.updateCheck = n.control("BUTTON", "Автоматически проверять обновления", 0x10003, 40, 548, 488, 28, 22)
	n.advancedButton = n.control("BUTTON", advancedTitle, 0x10000, 24, 516, 520, 32, 30)
	n.fields[0], n.auto, n.list, n.statusLabel = n.homeName, n.homeAuto, n.homeList, n.homeStatus
	n.updateWindow = n.panel(class, "Обновление Clipare", 520, 320, logicalRect{24, 24, 472, 216})
	n.window = n.updateWindow
	n.updateText = n.control("STATIC", "", 0, 40, 40, 440, 184, 0)
	n.updateInstall = n.control("BUTTON", "Обновить", 0x10000, 332, 264, 164, 36, 20)
	n.updateDismiss = n.control("BUTTON", "Понятно", 0x10000, 24, 264, 164, 36, 21)
	n.installDiscovery()
	n.installPairing()
	n.notice = n.panel(class, "Clipare", 568, 360)
	n.window = n.notice
	n.heading("Clipare", 24, 16, 440, 1)
	n.noticeText = n.control("STATIC", "", 0, 24, 72, 520, 200, 0)
	n.button("Понятно", 384, 304, 160, 31)
	n.window = n.home
}
func (n *nativeDesktop) addTray() {
	data := notifyIcon{Window: n.main, ID: 1, Flags: 7, Callback: 0x8001, Icon: n.icon}
	data.Size = uint32(unsafe.Sizeof(data))
	copy(data.Tip[:], syscall.StringToUTF16("Clipare"))
	shell.NewProc("Shell_NotifyIconW").Call(0, uintptr(unsafe.Pointer(&data)))
}
func (n *nativeDesktop) menu() {
	menu := call("CreatePopupMenu")
	defer call("DestroyMenu", menu)
	add := func(title string, id int, disabled bool) {
		flags := uintptr(0)
		if disabled {
			flags = 3
		}
		call("AppendMenuW", menu, flags, uintptr(id), uintptr(unsafe.Pointer(wide(title))))
	}
	add("Clipare", 0, true)
	add(n.state, 0, false)
	if bitmap := n.menuStatusIcon(menu); bitmap != 0 {
		defer gcall("DeleteObject", bitmap)
	}
	call("AppendMenuW", menu, 0x800, 0, 0)
	for _, p := range strings.Split(n.peerLines, "\n") {
		if p != "" {
			add(p, 0, true)
		}
	}
	call("AppendMenuW", menu, 0x800, 0, 0)
	text := "Возобновить синхронизацию"
	if n.enabled {
		text = "Приостановить синхронизацию"
	}
	add(text, eventPause, false)
	add("Добавить устройство", eventAdd, false)
	add("Открыть Clipare", eventSettings, false)
	add("Проверить обновления", eventCheckUpdate, false)
	add("Выйти", eventQuit, false)
	var pos struct{ X, Y int32 }
	call("GetCursorPos", uintptr(unsafe.Pointer(&pos)))
	call("SetForegroundWindow", n.main)
	id := call("TrackPopupMenu", menu, 0x102, uintptr(pos.X), uintptr(pos.Y), 0, n.window, 0)
	if id != 0 {
		n.events = append(n.events, int(id))
	}
	call("PostMessageW", n.main, 0, 0, 0)
}
func (n *nativeDesktop) Poll() int {
	var m message
	for call("PeekMessageW", uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1) != 0 {
		root := call("GetAncestor", m.Window, 2)
		if root == n.main || root == 0 {
			root = n.activePanel()
		}
		if m.ID == 0x100 && m.WParam == 27 {
			if root == n.window && n.comboOpen() {
				call("TranslateMessage", uintptr(unsafe.Pointer(&m)))
				call("DispatchMessageW", uintptr(unsafe.Pointer(&m)))
				continue
			}
			if root == n.updateWindow {
				if call("IsWindowVisible", n.updateDismiss) != 0 {
					call("SendMessageW", n.updateDismiss, 0xf5, 0, 0)
				}
			} else {
				call("SendMessageW", root, 0x10, 0, 0)
			}
			continue
		}
		if m.ID == 0x100 && m.WParam == 13 {
			if root == n.window && n.comboOpen() {
				call("TranslateMessage", uintptr(unsafe.Pointer(&m)))
				call("DispatchMessageW", uintptr(unsafe.Pointer(&m)))
				continue
			}
			button := call("GetFocus")
			if c, ok := n.controls[button]; !ok || c.class != "BUTTON" || call("GetWindowLongPtrW", button, ^uintptr(15))&15 == 3 {
				button = call("GetDlgItem", root, 1)
				if root == n.home && (call("GetFocus") == n.homeName || !n.navigation.AdvancedExpanded) {
					button = call("GetDlgItem", root, eventDeviceName)
				}
				switch n.navigation.View {
				case ViewUpdate:
					button = n.updateInstall
					if call("IsWindowVisible", button) == 0 {
						button = n.updateDismiss
					}
				case ViewPairing:
					if call("GetFocus") != n.homeName {
						button = n.pairAllow
					}
				case ViewDiscovery:
					if call("GetFocus") != n.homeName {
						button = call("GetDlgItem", root, 13)
					}
				}
			}
			if call("IsWindowVisible", button) != 0 && call("IsWindowEnabled", button) != 0 {
				call("SendMessageW", button, 0xf5, 0, 0)
			}
			continue
		}
		if call("IsDialogMessageW", root, uintptr(unsafe.Pointer(&m))) == 0 {
			call("TranslateMessage", uintptr(unsafe.Pointer(&m)))
			call("DispatchMessageW", uintptr(unsafe.Pointer(&m)))
		}
		n.revealFocus()
	}
	if len(n.events) == 0 {
		return 0
	}
	e := n.events[0]
	n.events = n.events[1:]
	return e
}
func (n *nativeDesktop) set(i int, s string) {
	call("SetWindowTextW", n.fields[i], uintptr(unsafe.Pointer(wide(s))))
}
func (n *nativeDesktop) Show(f form, peers []config.Peer, addresses []string) {
	call("SendMessageW", n.fields[2], 0x14B, 0, 0)
	for _, a := range addresses {
		call("SendMessageW", n.fields[2], 0x143, 0, uintptr(unsafe.Pointer(wide(a))))
	}
	for i, v := range f.Values {
		n.set(i, v)
	}
	auto := uintptr(0)
	if f.Autostart {
		auto = 1
	}
	call("SendMessageW", n.auto, 0xF1, auto, 0)
	call("SetWindowTextW", n.homeName, uintptr(unsafe.Pointer(wide(f.Values[0]))))
	call("SendMessageW", n.homeAuto, 0xF1, auto, 0)
	n.setRows(n.homeList, pairedRows(peers, nil))
	n.showDevicePresentation(len(peers))
	n.navigate(n.navigation.View)
	n.peerLines = ""
	n.Actions(actionMask(f))
}
func (n *nativeDesktop) Read() form {
	var f form
	for i, h := range n.fields {
		if i == 0 && call("IsWindowVisible", n.home) != 0 && call("IsWindowVisible", n.window) == 0 {
			h = n.homeName
		}
		size := call("GetWindowTextLengthW", h)
		if size > 8192 {
			size = 8192
		}
		b := make([]uint16, size+1)
		call("GetWindowTextW", h, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
		f.Values[i] = syscall.UTF16ToString(b)
	}
	f.Autostart = call("SendMessageW", n.auto, 0xF0, 0, 0) == 1
	if call("IsWindowVisible", n.home) != 0 && call("IsWindowVisible", n.window) == 0 {
		f.Autostart = call("SendMessageW", n.homeAuto, 0xF0, 0, 0) == 1
	}
	return f
}
func (n *nativeDesktop) Selected() int {
	list := n.homeList
	if call("IsWindowVisible", n.window) != 0 {
		list = n.list
	}
	return int(int32(call("SendMessageW", list, 0x188, 0, 0)))
}
func (n *nativeDesktop) SetPeer(p config.Peer) {
	for i, s := range []string{p.Name, p.ID, p.Address, strconv.Itoa(p.Port)} {
		n.set(i+6, s)
	}
	n.Actions(actionMask(n.Read()))
}
func (n *nativeDesktop) Update(status string, enabled bool, peers []config.Peer, states map[string]bool) {
	n.showDevicePresentation(len(peers))
	lines := peerStatuses(peers, states)
	if lines != n.peerLines {
		n.setRows(n.homeList, pairedRows(peers, states))
	}
	n.peerLines = lines
	text := "Возобновить"
	if enabled {
		text = "Пауза"
	}
	if windowText(n.pauseButton) != text {
		call("SetWindowTextW", n.pauseButton, uintptr(unsafe.Pointer(wide(text))))
	}
}
func (n *nativeDesktop) Alert(s string) {
	call("MessageBoxW", n.main, uintptr(unsafe.Pointer(wide(s))), uintptr(unsafe.Pointer(wide("Clipare"))), 0x40)
}
func (n *nativeDesktop) Actions(mask uint64) {
	for h, id := range n.actionButtons {
		value := uintptr(0)
		if mask&(1<<id) != 0 {
			value = 1
		}
		if call("IsWindowEnabled", h) != value {
			call("EnableWindow", h, value)
		}
	}
}

func (n *nativeDesktop) ApplyVisibility(mask uint64) {
	if n.applyVisible == mask {
		return
	}
	n.applyVisible = mask
	for h, id := range n.actionButtons {
		if id == eventSave || id == eventDeviceName {
			visible(h, mask&(1<<id) != 0 && (id != eventSave || n.navigation.AdvancedExpanded))
		}
	}
}
func (n *nativeDesktop) Close() {
	data := notifyIcon{Window: n.main, ID: 1}
	data.Size = uint32(unsafe.Sizeof(data))
	shell.NewProc("Shell_NotifyIconW").Call(2, uintptr(unsafe.Pointer(&data)))
	call("DestroyWindow", n.main)
	for _, icon := range n.statusIcons {
		if icon != 0 {
			call("DestroyIcon", icon)
		}
	}
	for _, font := range n.fonts {
		gcall("DeleteObject", font)
	}
	for _, brush := range []uintptr{n.surfaceBrush, n.backgroundBrush} {
		if brush != 0 {
			gcall("DeleteObject", brush)
		}
	}
	call("UnregisterClassW", uintptr(unsafe.Pointer(wide("ClipareSettingsWindow"))), n.instance)
}
func (n *nativeDesktop) Discovered(lines, status string) {
	rows := discoveredRows(lines)
	n.setRows(n.foundList, rows)
	show := uintptr(0)
	showList := uintptr(5)
	if len(rows) == 0 {
		show = 5
		showList = 0
	}
	call("ShowWindow", n.emptyDiscovery, show)
	call("ShowWindow", n.foundList, showList)
	enabled := uintptr(0)
	if len(rows) > 0 {
		enabled = 1
	}
	call("EnableWindow", call("GetDlgItem", n.found, 13), enabled)
	call("SetWindowTextW", n.foundStatus, uintptr(unsafe.Pointer(wide(status))))
	n.navigate(ViewDiscovery)
}
func (n *nativeDesktop) DiscoveredSelected() int {
	return int(int32(call("SendMessageW", n.foundList, 0x188, 0, 0)))
}
func (n *nativeDesktop) Pair(name, sas string, mode int) {
	incoming := mode == 1
	if incoming && n.navigation.View != ViewPairing {
		n.notifyPair(name)
	}
	n.pairIncoming = incoming
	n.pairMode = mode
	call("SetWindowTextW", n.pairName, uintptr(unsafe.Pointer(wide(name))))
	call("SetWindowTextW", n.pairCode, uintptr(unsafe.Pointer(wide(sas))))
	text := "Сравните код на другом компьютере и разрешите подключение там. Ожидание подтверждения…"
	show := uintptr(0)
	tag := uintptr(16)
	title := "Отменить"
	if incoming {
		text = "Это устройство хочет подключиться. Сравните коды на обоих компьютерах. Если они отличаются — отклоните подключение."
		show = 5
		tag = 15
		title = "Отклонить"
	}
	allowTag := uintptr(14)
	allowTitle := "Разрешить"
	if mode == 2 {
		show = 5
		allowTag = 18
		allowTitle = "Код совпадает"
		text = "Сравните коды на обоих компьютерах и нажмите «Код совпадает». На другом устройстве также разрешите подключение."
	}
	call("SetWindowLongPtrW", n.pairAllow, ^uintptr(11), allowTag)
	call("SetWindowTextW", n.pairAllow, uintptr(unsafe.Pointer(wide(allowTitle))))
	call("ShowWindow", n.pairAllow, show)
	call("SetWindowLongPtrW", n.pairReject, ^uintptr(11), tag)
	call("SetWindowTextW", n.pairReject, uintptr(unsafe.Pointer(wide(title))))
	call("SetWindowTextW", n.pairHelp, uintptr(unsafe.Pointer(wide(text))))
	n.navigate(ViewPairing)
}
func (n *nativeDesktop) PairClose() {
	if n.navigation.View == ViewNotice {
		n.navigation.End(ViewPairing)
	}
	if n.navigation.View == ViewPairing {
		n.navigate(ViewHome)
	}
}
func (n *nativeDesktop) UpdateSettings(version string, enabled bool) {
	v := uintptr(0)
	if enabled {
		v = 1
	}
	call("SendMessageW", n.updateCheck, 0xF1, v, 0)
	call("SetWindowTextW", n.updateVersion, uintptr(unsafe.Pointer(wide("Версия: "+version))))
}
func (n *nativeDesktop) UpdateEnabled() bool {
	return call("SendMessageW", n.updateCheck, 0xF0, 0, 0) == 1
}
func (n *nativeDesktop) Preferences(autostart, updates bool) {
	for h, on := range map[uintptr]bool{n.homeAuto: autostart, n.updateCheck: updates} {
		v := uintptr(0)
		if on {
			v = 1
		}
		call("SendMessageW", h, 0xf1, v, 0)
		call("InvalidateRect", h, 0, 1)
	}
}
func (n *nativeDesktop) UpdatePrompt(v updatePrompt) {
	call("SetWindowTextW", n.updateText, uintptr(unsafe.Pointer(wide(v.Text))))
	call("SetWindowTextW", n.updateInstall, uintptr(unsafe.Pointer(wide(v.Primary))))
	call("SetWindowLongPtrW", n.updateInstall, ^uintptr(11), uintptr(v.Action))
	call("SetWindowTextW", n.updateDismiss, uintptr(unsafe.Pointer(wide(v.Dismiss))))
	show := uintptr(0)
	if v.Primary != "" {
		show = 5
	}
	call("ShowWindow", n.updateInstall, show)
	show = 0
	if v.Dismiss != "" {
		show = 5
	}
	call("ShowWindow", n.updateDismiss, show)
	n.navigate(ViewUpdate)
}
func (n *nativeDesktop) UpdateClose() {
	if n.navigation.View == ViewNotice {
		n.navigation.End(ViewUpdate)
	}
	if n.navigation.View == ViewUpdate {
		n.navigate(ViewHome)
	}
}
