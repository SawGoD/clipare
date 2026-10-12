//go:build windows

package ui

import (
	"clipare/internal/config"
	"strings"
	"unsafe"
)

const (
	historyRowHeight  = 92
	historyListID     = 121
	historyComboID    = 142
	historyMaxID      = 141
	historyButtonBase = 9400
)

// History lives inside the home device card, exactly like discovery and
// pairing flows: same single-window navigation, no extra window.
func (n *nativeDesktop) installHistory() {
	n.window = n.home
	before := make(map[uintptr]bool)
	for h := range n.controls {
		before[h] = true
	}
	n.iconButton("Вернуться к устройствам", "\ue72b", 40, 224, 36, 31)
	n.heading("История", 88, 224, 440, 2)
	n.emptyHistory = n.control("STATIC", historyEmptyTitle+"\n"+historyEmptyHint, 0x4000, 40, 272, 488, 72, 0)
	n.historyStatus = n.control("STATIC", "", 0x4000, 40, 272, 488, 72, 0)
	n.setRole(n.historyStatus, 4)
	n.historyList = n.control("LISTBOX", "", 0x00210151, 40, 264, 488, 180, historyListID)
	call("SendMessageW", n.historyList, 0x1a0, 0, uintptr(n.scaledRect(n.home, logicalRect{h: historyRowHeight}).Bottom))
	n.historyButtons = nil
	for h, c := range n.controls {
		if !before[h] && c.parent == n.home {
			n.historyControls = append(n.historyControls, h)
			visible(h, false)
		}
	}
}

func (n *nativeDesktop) installHistorySettings() {
	n.window = n.home
	before := make(map[uintptr]bool)
	for h := range n.controls {
		before[h] = true
	}
	n.historyCheck = n.control("BUTTON", "Сохранять историю", 0x10003, 40, 588, 488, 28, eventHistoryEnabled)
	n.label("Хранить", 40, 622, 236)
	n.historyRetention = n.control("COMBOBOX", "", 0x00210213, 40, 640, 236, 200, historyComboID)
	n.styleCombo(n.historyRetention)
	n.label("Максимум записей", 292, 622, 236)
	n.fields[11] = n.control("EDIT", "", 0x00810080, 292, 640, 236, 26, historyMaxID)
	n.helpButton(144, 492, 640, "Записи удаляются автоматически: по сроку хранения и по достижении максимума. История хранится только на этом устройстве и шифруется.")
	n.historyClear = n.control("BUTTON", "Очистить историю", 0x10000, 40, 678, 488, 32, eventHistoryClear)
	n.historyBase = map[uintptr]int{}
	for h, c := range n.controls {
		if !before[h] && c.parent == n.home {
			n.historySettings = append(n.historySettings, h)
			n.historyBase[h] = c.bounds.y
			visible(h, false)
		}
	}
}

func (n *nativeDesktop) ShowHistory(rows []historyRow, keyLost bool) {
	n.historyRows = rows
	accessibility := make([]string, len(rows))
	for i, row := range rows {
		accessibility[i] = historySummary(row)
	}
	selection := int(int32(call("SendMessageW", n.historyList, 0x188, 0, 0)))
	wasVisible := call("GetWindowLongPtrW", n.historyList, ^uintptr(15))&0x10000000 != 0
	call("SendMessageW", n.historyList, 0xb, 0, 0)
	call("SendMessageW", n.historyList, 0x184, 0, 0)
	for _, text := range accessibility {
		call("SendMessageW", n.historyList, 0x180, 0, uintptr(unsafe.Pointer(wide(text))))
	}
	if selection >= 0 && selection < len(rows) {
		call("SendMessageW", n.historyList, 0x186, uintptr(selection), 0)
	}
	call("SendMessageW", n.historyList, 0xb, 1, 0)
	visible(n.historyList, wasVisible && len(rows) > 0)
	call("InvalidateRect", n.historyList, 0, 1)
	status := ""
	if keyLost {
		status = historyKeyLostText
	}
	if windowText(n.historyStatus) != status {
		call("SetWindowTextW", n.historyStatus, uintptr(unsafe.Pointer(wide(status))))
	}
	n.layoutHistory()
	n.navigate(ViewHistory)
}

func (n *nativeDesktop) HistoryEnabled() bool {
	return call("SendMessageW", n.historyCheck, 0xF0, 0, 0) == 1
}

func (n *nativeDesktop) HistoryPrefs(c config.History) {
	enabled := uintptr(0)
	if c.Enabled {
		enabled = 1
	}
	call("SendMessageW", n.historyCheck, 0xF1, enabled, 0)
	n.selectRetention(c.Retention)
	call("SetWindowTextW", n.fields[11], uintptr(unsafe.Pointer(wide(intLabel(c.NormalizedMaxItems())))))
	call("InvalidateRect", n.historyCheck, 0, 1)
}

func (n *nativeDesktop) selectRetention(duration string) {
	label := RetentionToLabel(duration)
	if label == "" {
		call("SendMessageW", n.historyRetention, 0x14E, ^uintptr(0), 0) // CB_SETCURSEL -1
		return
	}
	call("SendMessageW", n.historyRetention, 0x146, 0, uintptr(unsafe.Pointer(wide(label)))) // CB_SELECTSTRING
}

func intLabel(v int) string {
	if v == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

// layoutHistory mirrors layoutDiscovery sizing for the shared device card.
func (n *nativeDesktop) layoutHistory() int {
	show := n.navigation.View == ViewHistory
	for _, h := range n.historyControls {
		visible(h, show)
	}
	if !show {
		return 0
	}
	count := len(n.historyRows)
	height := count * historyRowHeight
	if height > 276 {
		height = 276
	}
	if count == 0 {
		height = 88
	}
	visible(n.historyList, count > 0)
	visible(n.emptyHistory, count == 0)
	status := windowText(n.historyStatus)
	visible(n.historyStatus, count == 0 && status != "")
	visible(n.emptyHistory, count == 0 && status == "")
	c := n.controls[n.historyList]
	c.bounds.h = height
	n.controls[n.historyList] = c
	c = n.controls[n.emptyHistory]
	c.bounds.h = height
	n.controls[n.emptyHistory] = c
	if count == 0 {
		return 104 + height - 228
	}
	return 120 + height - 228
}

// updateHistoryButtons keeps one pool of per-row icon buttons positioned over
// the list's actual item rectangles.
func (n *nativeDesktop) updateHistoryButtons() {
	rows := n.historyRows
	need := len(rows) * 4
	if len(n.historyButtons) < need {
		previous := n.window
		n.window = n.home
		for len(n.historyButtons) < need {
			index := len(n.historyButtons)
			kind := index % 4
			label, glyph := "Копировать", "\ue8c8"
			switch kind {
			case 1:
				label, glyph = "Показать значение", "\ue7b3"
			case 2:
				label, glyph = "Открыть ссылку", "\ue8a7"
			case 3:
				label, glyph = "Удалить запись", "\ue74d"
			}
			h := n.iconButton(label, glyph, 0, 0, 32, historyButtonBase+index)
			n.historyButtons = append(n.historyButtons, h)
		}
		n.window = previous
	}
	n.positionHistoryButtons()
}

func (n *nativeDesktop) positionHistoryButtons() {
	var client winRect
	call("GetClientRect", n.home, uintptr(unsafe.Pointer(&client)))
	for index, h := range n.historyButtons {
		row, kind := index/4, index%4
		if n.navigation.View != ViewHistory || row >= len(n.historyRows) {
			visible(h, false)
			continue
		}
		row0 := n.historyRows[row]
		switch {
		case row0.group:
			visible(h, false)
			continue
		case kind == 1 && !row0.sensitive, kind == 2 && !row0.url:
			visible(h, false)
			continue
		}
		var r winRect
		if int32(call("SendMessageW", n.historyList, 0x198, uintptr(row), uintptr(unsafe.Pointer(&r)))) < 0 || r.Top < 0 || r.Bottom > client.Bottom {
			visible(h, false)
			continue
		}
		dpi := int(call("GetDpiForWindow", n.home))
		if dpi == 0 {
			dpi = 96
		}
		size := int32(32 * dpi / 96)
		x := client.Right - size - 8 - int32(kind)*int32(float64(38)*n.scale)
		y := r.Top + (r.Bottom-r.Top-size)/2 + int32(12*n.scale)
		point := struct{ X, Y int32 }{x, y}
		call("MapWindowPoints", n.historyList, n.home, uintptr(unsafe.Pointer(&point)), 1)
		call("SetWindowPos", h, 0, uintptr(point.X), uintptr(point.Y), uintptr(size), uintptr(size), 0x10)
		call("SendMessageW", h, 0x30, n.fontFor(n.home, 5), 0)
		glyph := "\ue8c8"
		switch kind {
		case 1:
			glyph = "\ue7b3"
			if row0.revealed {
				glyph = "\ue7b2"
			}
		case 2:
			glyph = "\ue8a7"
		case 3:
			glyph = "\ue74d"
		}
		if n.icons[h] != glyph {
			n.icons[h] = glyph
			call("InvalidateRect", h, 0, 1)
		}
		visible(h, true)
	}
}

// historyRowAction maps a pooled row button click to a loop event or a direct
// system action (opening the URL through the shell handler).
func (n *nativeDesktop) historyRowAction(row, kind int) {
	if row < 0 || row >= len(n.historyRows) {
		return
	}
	target := n.historyRows[row]
	if target.group {
		return
	}
	switch kind {
	case 0:
		n.events = append(n.events, eventHistoryCopy)
	case 1:
		if target.sensitive {
			n.events = append(n.events, eventHistoryEye)
		}
	case 2:
		// Classification guarantees http(s); double-check before the shell.
		if target.url && (strings.HasPrefix(target.link, "http://") || strings.HasPrefix(target.link, "https://")) {
			shell.NewProc("ShellExecuteW").Call(0,
				uintptr(unsafe.Pointer(wide("open"))),
				uintptr(unsafe.Pointer(wide(target.link))), 0, 0, 5)
		}
	case 3:
		n.events = append(n.events, eventHistoryDelete)
	}
}
