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
	pairFont                                                                           uintptr
	theme                                                                              winTheme
	surfaceBrush, backgroundBrush                                                      uintptr
	windows                                                                            map[uintptr]winWindow
	controls                                                                           map[uintptr]winControl
	fonts                                                                              map[[2]int]uintptr
	rows                                                                               map[uintptr][]deviceRow
	pauseButton, emptyDevices, emptyDiscovery                                          uintptr
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
		case 1, 8, 13, 14, 18, 20:
			kind = 1
		case 9:
			kind = 2
		}
	}
	r := n.scaledRect(n.window, logicalRect{x, y, w, h})
	handle := call("CreateWindowExW", 0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(text))), style|0x50000000, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), n.window, uintptr(id), n.instance, 0)
	n.controls[handle] = winControl{parent: n.window, class: class, bounds: logicalRect{x, y, w, h}, kind: kind}
	call("SendMessageW", handle, 0x30, n.fontFor(n.window, 0), 1)
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
	call("SetProcessDpiAwarenessContext", ^uintptr(3)) // per-monitor v2
	if dpi := call("GetDpiForSystem"); dpi > 0 {
		n.scale = float64(dpi) / 96
	}
	n.instance, _, _ = k32.NewProc("GetModuleHandleW").Call(0)
	n.windows = make(map[uintptr]winWindow)
	n.controls = make(map[uintptr]winControl)
	n.fonts = make(map[[2]int]uintptr)
	n.rows = make(map[uintptr][]deviceRow)
	n.applyTheme()
	n.icon = call("LoadIconW", 0, 32516)
	n.taskbar = uint32(call("RegisterWindowMessageW", uintptr(unsafe.Pointer(wide("TaskbarCreated")))))
	n.callback = syscall.NewCallback(func(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
		if msg == n.taskbar && n.taskbar != 0 {
			n.addTray()
			return 0
		}
		if r, ok := n.themeMessage(hwnd, msg, w, l); ok {
			return r
		}
		switch msg {
		case 0x2e0:
			n.dpiChanged(hwnd, w, l)
			return 0
		case 0x10:
			if hwnd == n.found {
				n.events = append(n.events, 17)
			}
			if hwnd == n.pair {
				if n.pairIncoming {
					n.events = append(n.events, 15)
				} else {
					n.events = append(n.events, 16)
				}
			}
			call("ShowWindow", hwnd, 0)
			return 0
		case 0x16:
			if w != 0 {
				n.events = append(n.events, eventQuit)
			}
			return 0
		case 0x111:
			id := int(w & 0xffff)
			if id == 30 {
				call("ShowWindow", n.home, 0)
				call("ShowWindow", n.window, 5)
				call("SetForegroundWindow", n.window)
				return 0
			}
			if id == 1 && hwnd == n.home {
				n.set(0, windowText(n.homeName))
				call("SendMessageW", n.auto, 0xF1, call("SendMessageW", n.homeAuto, 0xF0, 0, 0), 0)
			}
			if id == 120 && w>>16 == 1 {
				n.events = append(n.events, eventSelect)
			} else if id >= 1 && id <= 22 {
				n.events = append(n.events, id)
			}
			return 0
		case 0x8001:
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
	n.window = n.panel(class, "Дополнительно — Clipare", 848, 690, logicalRect{24, 64, 376, 526}, logicalRect{416, 64, 408, 526})
	if n.window == 0 {
		return errors.New("Не удалось создать окно Clipare")
	}
	n.heading("Дополнительно", 24, 16, 600, 1)
	n.heading("Этот компьютер", 40, 80, 344, 2)
	n.label("Имя устройства", 40, 122, 344)
	n.input(0, 40, 150, 344, false)
	n.label("Device ID", 40, 194, 344)
	n.input(1, 40, 222, 344, false)
	call("SendMessageW", n.fields[1], 0xCF, 1, 0)
	n.label("NetBird IP", 40, 266, 236)
	n.fields[2] = n.control("COMBOBOX", "", 0x00210042, 40, 294, 236, 200, 102)
	n.label("Порт", 288, 266, 96)
	n.input(3, 288, 294, 96, false)
	n.label("Legacy общий ключ", 40, 338, 344)
	n.input(4, 40, 366, 344, true)
	n.button("Создать новый ключ", 40, 410, 240, 5)
	n.button("Скопировать код подключения", 40, 454, 344, 6)
	n.auto = n.control("BUTTON", "Запускать при входе в систему", 0x10003, 40, 508, 344, 28, 121)
	n.statusLabel = n.control("STATIC", "Настройте подключение", 0, 40, 548, 344, 30, 0)
	n.heading("Ручное подключение", 432, 80, 376, 2)
	n.list = n.control("LISTBOX", "", 0x00210001, 432, 122, 376, 100, 120)
	n.label("Имя устройства", 432, 238, 376)
	n.input(6, 432, 266, 376, false)
	n.label("Device ID", 432, 310, 376)
	n.input(7, 432, 338, 376, false)
	n.label("IP или имя NetBird", 432, 382, 264)
	n.input(8, 432, 410, 264, false)
	n.label("Порт", 708, 382, 100)
	n.input(9, 708, 410, 100, false)
	n.button("Добавить / изменить", 432, 454, 236, 8)
	n.button("Удалить", 684, 454, 124, 9)
	n.label("Legacy код подключения", 432, 498, 376)
	n.input(10, 432, 526, 216, true)
	n.button("Добавить по коду", 660, 526, 148, 7)
	n.fields[5] = n.control("STATIC", "", 0x4000, 24, 606, 800, 22, 0)
	n.setRole(n.fields[5], 4)
	n.label("Код содержит общий ключ. Передавайте его приватно.", 24, 642, 544)
	n.button("Сохранить настройки", 584, 638, 240, 1)
	n.state = "Настройте подключение"
	n.installHome(class)
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
	advanced := n.window
	n.home = n.panel(class, "Clipare", 568, 688, logicalRect{24, 64, 520, 128}, logicalRect{24, 208, 520, 228}, logicalRect{24, 452, 520, 68}, logicalRect{24, 536, 520, 136})
	n.window = n.home
	n.heading("Clipare", 24, 16, 300, 1)
	n.button("Дополнительно…", 360, 16, 184, 30)
	n.heading("Этот компьютер", 40, 80, 280, 2)
	n.homeName = n.control("EDIT", "", 0x00810080, 40, 116, 332, 30, 130)
	n.button("Сохранить", 384, 116, 144, 1)
	n.homeStatus = n.control("STATIC", "Ожидание NetBird", 0x4000, 40, 158, 312, 22, 0)
	n.pauseButton = n.control("BUTTON", "Пауза", 0x10000, 384, 152, 144, 32, eventPause)
	n.heading("Устройства", 40, 224, 360, 2)
	n.homeList = n.deviceList(40, 264, 488, 120, 120)
	n.emptyDevices = n.control("STATIC", "Пока нет подключённых устройств", 0x4000, 56, 290, 456, 24, 0)
	n.button("+ Добавить устройство", 40, 392, 268, 11)
	n.button("Удалить", 400, 392, 128, 9)
	n.homeAuto = n.control("BUTTON", "Запускать при входе в систему", 0x10003, 40, 472, 488, 28, 131)
	n.heading("Обновления", 40, 552, 360, 2)
	n.updateCheck = n.control("BUTTON", "Автоматически проверять обновления", 0x10003, 40, 588, 488, 28, 22)
	n.updateVersion = n.control("STATIC", "", 0x4000, 40, 638, 232, 24, 0)
	n.button("Проверить обновления", 292, 632, 236, 19)
	n.updateWindow = n.panel(class, "Обновление Clipare", 520, 320, logicalRect{24, 24, 472, 216})
	n.window = n.updateWindow
	n.updateText = n.control("STATIC", "", 0, 40, 40, 440, 184, 0)
	n.updateInstall = n.control("BUTTON", "Обновить", 0x10000, 332, 264, 164, 36, 20)
	n.updateDismiss = n.control("BUTTON", "Понятно", 0x10000, 24, 264, 164, 36, 21)
	n.found = n.panel(class, "Добавить устройство", 568, 420, logicalRect{24, 64, 520, 248})
	n.window = n.found
	n.heading("Найденные устройства", 24, 16, 520, 1)
	n.foundList = n.deviceList(40, 80, 488, 180, 132)
	n.foundStatus = n.control("STATIC", "Поиск устройств…", 0, 40, 268, 488, 38, 0)
	n.setRole(n.foundStatus, 4)
	n.emptyDiscovery = n.control("STATIC", "Устройства Clipare не найдены\r\n\r\nУбедитесь, что NetBird запущен\r\nна обоих компьютерах.", 1, 64, 126, 440, 96, 0)
	n.button("Обновить список", 24, 332, 220, 12)
	n.button("Подключить", 324, 332, 220, 13)
	n.button("Не нашли? Добавить по коду…", 24, 376, 340, 30)
	n.pair = n.panel(class, "Подключение устройства", 520, 380, logicalRect{24, 112, 472, 96})
	n.window = n.pair
	n.heading("Подключение устройства", 24, 16, 472, 1)
	n.pairName = n.control("STATIC", "", 0x4000, 24, 60, 472, 28, 0)
	n.heading("Код проверки", 40, 122, 440, 2)
	n.pairCode = n.control("STATIC", "", 1, 40, 156, 440, 48, 0)
	n.setRole(n.pairCode, 3)
	n.pairHelp = n.control("STATIC", "", 0, 24, 232, 472, 80, 0)
	n.pairAllow = n.control("BUTTON", "Разрешить", 0x10000, 316, 328, 180, 36, 14)
	n.pairReject = n.control("BUTTON", "Отклонить", 0x10000, 24, 328, 164, 36, 15)
	n.window = advanced
}
func (n *nativeDesktop) addTray() {
	data := notifyIcon{Window: n.window, ID: 1, Flags: 7, Callback: 0x8001, Icon: n.icon}
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
	add(n.state, 0, true)
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
	add("Настройки…", eventSettings, false)
	add("Проверить обновления", eventCheckUpdate, false)
	add("Выйти", eventQuit, false)
	var pos struct{ X, Y int32 }
	call("GetCursorPos", uintptr(unsafe.Pointer(&pos)))
	call("SetForegroundWindow", n.window)
	id := call("TrackPopupMenu", menu, 0x102, uintptr(pos.X), uintptr(pos.Y), 0, n.window, 0)
	if id != 0 {
		n.events = append(n.events, int(id))
	}
	call("PostMessageW", n.window, 0, 0, 0)
}
func (n *nativeDesktop) Poll() int {
	var m message
	for call("PeekMessageW", uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1) != 0 {
		root := call("GetAncestor", m.Window, 2)
		if root == 0 {
			root = n.window
		}
		if root == n.updateWindow && m.ID == 0x100 && (m.WParam == 13 || m.WParam == 27) {
			button := n.updateDismiss
			if m.WParam == 13 && call("IsWindowVisible", n.updateInstall) != 0 {
				button = n.updateInstall
			}
			if call("IsWindowVisible", button) != 0 {
				call("SendMessageW", button, 0xF5, 0, 0)
			}
			continue
		}
		if call("IsDialogMessageW", root, uintptr(unsafe.Pointer(&m))) == 0 {
			call("TranslateMessage", uintptr(unsafe.Pointer(&m)))
			call("DispatchMessageW", uintptr(unsafe.Pointer(&m)))
		}
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
	call("SendMessageW", n.list, 0x184, 0, 0)
	for _, p := range peers {
		call("SendMessageW", n.list, 0x180, 0, uintptr(unsafe.Pointer(wide(peerLabel(p)))))
	}
	auto := uintptr(0)
	if f.Autostart {
		auto = 1
	}
	call("SendMessageW", n.auto, 0xF1, auto, 0)
	call("SetWindowTextW", n.homeName, uintptr(unsafe.Pointer(wide(f.Values[0]))))
	call("SendMessageW", n.homeAuto, 0xF1, auto, 0)
	n.setRows(n.homeList, pairedRows(peers, nil))
	showEmpty := uintptr(0)
	if len(peers) == 0 {
		showEmpty = 5
	}
	call("ShowWindow", n.emptyDevices, showEmpty)
	target := n.home
	if call("IsWindowVisible", n.window) != 0 {
		target = n.window
	}
	call("ShowWindow", target, 5)
	call("SetForegroundWindow", target)
	n.peerLines = ""
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
}
func (n *nativeDesktop) Update(status string, enabled bool, peers []config.Peer, states map[string]bool) {
	call("SetWindowTextW", n.statusLabel, uintptr(unsafe.Pointer(wide(status))))
	n.state = status
	n.enabled = enabled
	lines := peerStatuses(peers, states)
	if lines != n.peerLines {
		n.setRows(n.homeList, pairedRows(peers, states))
	}
	n.peerLines = lines
	call("SetWindowTextW", n.homeStatus, uintptr(unsafe.Pointer(wide(status))))
	text := "Возобновить"
	if enabled {
		text = "Пауза"
	}
	call("SetWindowTextW", n.pauseButton, uintptr(unsafe.Pointer(wide(text))))
}
func (n *nativeDesktop) Alert(s string) {
	call("MessageBoxW", n.window, uintptr(unsafe.Pointer(wide(s))), uintptr(unsafe.Pointer(wide("Clipare"))), 0x40)
}
func (n *nativeDesktop) Close() {
	data := notifyIcon{Window: n.window, ID: 1}
	data.Size = uint32(unsafe.Sizeof(data))
	shell.NewProc("Shell_NotifyIconW").Call(2, uintptr(unsafe.Pointer(&data)))
	call("DestroyWindow", n.window)
	call("DestroyWindow", n.home)
	call("DestroyWindow", n.found)
	call("DestroyWindow", n.pair)
	call("DestroyWindow", n.updateWindow)
	if n.pairFont != 0 {
		syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(n.pairFont)
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
	if len(rows) == 0 {
		show = 5
	}
	call("ShowWindow", n.emptyDiscovery, show)
	call("SetWindowTextW", n.foundStatus, uintptr(unsafe.Pointer(wide(status))))
	call("ShowWindow", n.found, 5)
	call("SetForegroundWindow", n.found)
}
func (n *nativeDesktop) DiscoveredSelected() int {
	return int(int32(call("SendMessageW", n.foundList, 0x188, 0, 0)))
}
func (n *nativeDesktop) Pair(name, sas string, mode int) {
	incoming := mode == 1
	n.pairIncoming = incoming
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
	call("ShowWindow", n.pair, 5)
	call("SetForegroundWindow", n.pair)
}
func (n *nativeDesktop) PairClose() { call("ShowWindow", n.pair, 0) }
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
	call("ShowWindow", n.updateWindow, 5)
	call("SetForegroundWindow", n.updateWindow)
}
func (n *nativeDesktop) UpdateClose() { call("ShowWindow", n.updateWindow, 0) }
