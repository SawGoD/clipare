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
	updateWindow, updateCheck, updateVersion, updateText, updateInstall         uintptr
	window, instance, callback, icon                                            uintptr
	fields                                                                      [11]uintptr
	auto, list                                                                  uintptr
	statusLabel                                                                 uintptr
	events                                                                      []int
	state                                                                       string
	enabled                                                                     bool
	peerLines                                                                   string
	scale                                                                       float64
	taskbar                                                                     uint32
	home, found, pair                                                           uintptr
	homeName, homeAuto, homeList, homeStatus                                    uintptr
	foundList, foundStatus, pairName, pairCode, pairHelp, pairAllow, pairReject uintptr
	pairIncoming                                                                bool
	pairFont                                                                    uintptr
}

func newDesktop() (desktop, error) { return &nativeDesktop{scale: 1}, nil }
func wide(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(s, "\x00", ""))
	return p
}
func call(name string, args ...uintptr) uintptr { r, _, _ := win.NewProc(name).Call(args...); return r }
func (n *nativeDesktop) px(v int) uintptr       { return uintptr(int(float64(v) * n.scale)) }
func (n *nativeDesktop) control(class, text string, style uintptr, x, y, w, h, id int) uintptr {
	handle := call("CreateWindowExW", 0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(text))), style|0x50000000, n.px(x), n.px(y), n.px(w), n.px(h), n.window, uintptr(id), n.instance, 0)
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("GetStockObject").Call(17)
	call("SendMessageW", handle, 0x30, font, 1)
	return handle
}
func (n *nativeDesktop) label(text string, x, y, w int) { n.control("STATIC", text, 0, x, y, w, 22, 0) }
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
	call("SetProcessDPIAware")
	if dpi := call("GetDpiForSystem"); dpi > 0 {
		n.scale = float64(dpi) / 96
	}
	n.instance, _, _ = k32.NewProc("GetModuleHandleW").Call(0)
	n.icon = call("LoadIconW", 0, 32516)
	n.taskbar = uint32(call("RegisterWindowMessageW", uintptr(unsafe.Pointer(wide("TaskbarCreated")))))
	n.callback = syscall.NewCallback(func(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
		if msg == n.taskbar && n.taskbar != 0 {
			n.addTray()
			return 0
		}
		switch msg {
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
	wc := windowClass{Proc: n.callback, Instance: n.instance, Icon: n.icon, Cursor: call("LoadCursorW", 0, 32512), Background: 6, Name: class}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if call("RegisterClassExW", uintptr(unsafe.Pointer(&wc))) == 0 {
		return errors.New("Не удалось зарегистрировать окно Clipare")
	}
	n.window = call("CreateWindowExW", 0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(wide("Настройки Clipare"))), 0x00CA0000, 0x80000000, 0x80000000, n.px(780), n.px(660), 0, 0, n.instance, 0)
	if n.window == 0 {
		return errors.New("Не удалось создать окно Clipare")
	}
	n.label("Этот компьютер", 24, 18, 320)
	n.label("Имя устройства", 24, 50, 320)
	n.input(0, 24, 74, 320, false)
	n.label("ID устройства (создан автоматически)", 24, 112, 320)
	n.input(1, 24, 136, 320, false)
	call("SendMessageW", n.fields[1], 0xCF, 1, 0)
	n.label("NetBird IP для подключения", 24, 174, 320)
	n.fields[2] = n.control("COMBOBOX", "", 0x00210042, 24, 198, 320, 200, 102)
	n.label("Порт", 24, 236, 320)
	n.input(3, 24, 260, 320, false)
	n.label("Legacy общий ключ", 24, 298, 320)
	n.input(4, 24, 322, 320, true)
	n.button("Создать новый ключ", 24, 363, 200, 5)
	n.button("Скопировать код подключения", 24, 407, 320, 6)
	n.auto = n.control("BUTTON", "Запускать при входе в систему", 0x10003, 24, 453, 330, 28, 121)
	n.statusLabel = n.control("STATIC", "Настройте подключение", 0, 24, 493, 330, 44, 0)
	n.label("Другие компьютеры", 390, 18, 340)
	n.list = n.control("LISTBOX", "", 0x00A10001, 390, 50, 340, 136, 120)
	n.label("Имя", 390, 198, 340)
	n.input(6, 390, 222, 340, false)
	n.label("ID другого устройства", 390, 260, 340)
	n.input(7, 390, 284, 340, false)
	n.label("IP или имя NetBird", 390, 322, 230)
	n.input(8, 390, 346, 235, false)
	n.label("Порт", 635, 322, 95)
	n.input(9, 635, 346, 95, false)
	n.button("Добавить / изменить", 390, 389, 220, 8)
	n.button("Удалить", 618, 389, 112, 9)
	n.label("Код подключения другого компьютера", 390, 434, 340)
	n.input(10, 390, 458, 340, true)
	n.button("Добавить по коду", 390, 498, 220, 7)
	n.fields[5] = n.control("STATIC", "", 0, 24, 542, 710, 22, 0)
	n.label("Код содержит общий ключ. Передавайте его приватно.", 24, 565, 710)
	n.button("Сохранить настройки", 500, 590, 230, 1)
	n.state = "Настройте подключение"
	n.installHome(class)
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
	panel := func(title string, h int) uintptr {
		return call("CreateWindowExW", 0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(wide(title))), 0x00CA0000, 0x80000000, 0x80000000, n.px(500), n.px(h), 0, 0, n.instance, 0)
	}
	n.home = panel("Clipare", 644)
	n.window = n.home
	n.label("Этот компьютер", 24, 20, 430)
	n.homeName = n.control("EDIT", "", 0x00810080, 24, 50, 432, 28, 130)
	n.homeStatus = n.control("STATIC", "Ожидание NetBird", 0, 24, 92, 432, 42, 0)
	n.label("Устройства", 24, 144, 432)
	n.homeList = n.control("LISTBOX", "", 0x00A10001, 24, 174, 432, 136, 120)
	n.button("+ Добавить устройство", 24, 324, 254, 11)
	n.button("Удалить", 334, 324, 122, 9)
	n.homeAuto = n.control("BUTTON", "Запускать при входе в систему", 0x10003, 24, 368, 432, 28, 131)
	n.button("Дополнительно…", 24, 416, 190, 30)
	n.button("Сохранить", 320, 416, 136, 1)
	n.label("Обновления", 24, 464, 432)
	n.updateCheck = n.control("BUTTON", "Автоматически проверять обновления", 0x10003, 24, 494, 432, 28, 22)
	n.updateVersion = n.control("STATIC", "", 0, 24, 548, 200, 24, 0)
	n.button("Проверить обновления", 254, 540, 202, 19)
	n.updateWindow = panel("Обновление Clipare", 320)
	n.window = n.updateWindow
	n.updateText = n.control("STATIC", "", 0, 24, 24, 432, 170, 0)
	n.updateInstall = n.control("BUTTON", "Обновить", 0x10000, 292, 220, 164, 32, 20)
	n.button("Позже", 24, 220, 164, 21)
	n.found = panel("Добавить устройство", 420)
	n.window = n.found
	n.label("Найденные устройства", 24, 20, 432)
	n.foundList = n.control("LISTBOX", "", 0x00A10001, 24, 50, 432, 180, 132)
	n.foundStatus = n.control("STATIC", "Поиск устройств…", 0, 24, 240, 432, 44, 0)
	n.button("Обновить", 24, 294, 126, 12)
	n.button("Подключить", 302, 294, 154, 13)
	n.button("Не нашли? Добавить по коду…", 24, 340, 310, 30)
	n.pair = panel("Подключение устройства", 340)
	n.window = n.pair
	n.pairName = n.control("STATIC", "", 0, 24, 20, 432, 36, 0)
	n.pairCode = n.control("STATIC", "", 1, 24, 74, 432, 56, 0)
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("CreateFontW").Call(n.px(38), 0, 0, 0, 500, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(wide("Segoe UI"))))
	n.pairFont = font
	call("SendMessageW", n.pairCode, 0x30, font, 1)
	n.pairHelp = n.control("STATIC", "", 0, 24, 150, 432, 70, 0)
	n.pairAllow = n.control("BUTTON", "Разрешить", 0x10000, 292, 244, 164, 32, 14)
	n.pairReject = n.control("BUTTON", "Отклонить", 0x10000, 24, 244, 164, 32, 15)
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
	call("SendMessageW", n.homeList, 0x184, 0, 0)
	for _, p := range peers {
		call("SendMessageW", n.homeList, 0x180, 0, uintptr(unsafe.Pointer(wide(peerLabel(p)))))
	}
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
		selection := call("SendMessageW", n.homeList, 0x188, 0, 0)
		call("SendMessageW", n.homeList, 0x184, 0, 0)
		for _, s := range strings.Split(lines, "\n") {
			if s != "" {
				call("SendMessageW", n.homeList, 0x180, 0, uintptr(unsafe.Pointer(wide(s))))
			}
		}
		call("SendMessageW", n.homeList, 0x186, selection, 0)
	}
	n.peerLines = lines
	call("SetWindowTextW", n.homeStatus, uintptr(unsafe.Pointer(wide(status))))
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
	call("UnregisterClassW", uintptr(unsafe.Pointer(wide("ClipareSettingsWindow"))), n.instance)
}
func (n *nativeDesktop) Discovered(lines, status string) {
	call("SendMessageW", n.foundList, 0x184, 0, 0)
	for _, s := range strings.Split(lines, "\n") {
		if s != "" {
			call("SendMessageW", n.foundList, 0x180, 0, uintptr(unsafe.Pointer(wide(s))))
		}
	}
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
func (n *nativeDesktop) UpdatePrompt(text string, installable bool) {
	call("SetWindowTextW", n.updateText, uintptr(unsafe.Pointer(wide(text))))
	show := uintptr(0)
	if installable {
		show = 5
	}
	call("ShowWindow", n.updateInstall, show)
	call("ShowWindow", n.updateWindow, 5)
	call("SetForegroundWindow", n.updateWindow)
}
func (n *nativeDesktop) UpdateClose() { call("ShowWindow", n.updateWindow, 0) }
