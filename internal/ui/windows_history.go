//go:build windows

package ui

import (
	"clipare/internal/config"
	"reflect"
	"strings"
	"syscall"
	"unsafe"
)

const (
	historyListID  = 121
	historyComboID = 142
	// n.input numbers fields as 100+index; the max-items field is fields[11].
	historyMaxID      = 111
	historyButtonBase = 9400
	// Event kinds: copy, eye, link, trash, expand. Visible slots are compacted.
	historyKindsPerRow = 5
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
	// Variable item heights: expanded rows grow with their content.
	n.historyList = n.control("LISTBOX", "", 0x00210161, 40, 264, 488, 180, historyListID)
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
	// Same style as the address combo: identical theming and frame rendering.
	n.historyRetention = n.control("COMBOBOX", "", 0x00210253, 40, 640, 236, 200, historyComboID)
	n.fields[12] = n.historyRetention
	n.styleCombo(n.historyRetention)
	for _, label := range RetentionLabels {
		call("SendMessageW", n.historyRetention, 0x143, 0, uintptr(unsafe.Pointer(wide(label)))) // CB_ADDSTRING
	}
	n.label("Максимум записей", 292, 622, 236)
	n.input(11, 292, 640, 236, false)
	n.historyClear = n.control("BUTTON", "Очистить историю", 0x10000, 40, 678, 444, 32, eventHistoryClear)
	n.helpButton(144, 492, 684, "Записи удаляются автоматически: по сроку хранения и по достижении максимума. История хранится только на этом устройстве и шифруется.")
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
	rows = append([]historyRow(nil), rows...)
	for i := range rows {
		if n.px(rows[i].height) > 255 {
			rows[i].height = int(255 / n.scale)
		}
	}
	status := ""
	if keyLost {
		status = historyKeyLostText
	}
	// Identical payloads are ignored: background refreshes must not blink.
	if reflect.DeepEqual(rows, n.historyRows) && windowText(n.historyStatus) == status && n.navigation.View == ViewHistory {
		return
	}
	selection := int(int32(call("SendMessageW", n.historyList, 0x188, 0, 0)))
	selectedID := ""
	if selection >= 0 && selection < len(n.historyRows) {
		selectedID = n.historyRows[selection].id
	}
	n.historyRows = rows
	accessibility := make([]string, len(rows))
	for i, row := range rows {
		accessibility[i] = historySummary(row)
	}
	wasVisible := call("GetWindowLongPtrW", n.historyList, ^uintptr(15))&0x10000000 != 0
	call("SendMessageW", n.historyList, 0xb, 0, 0)
	call("SendMessageW", n.historyList, 0x184, 0, 0)
	for i, text := range accessibility {
		call("SendMessageW", n.historyList, 0x180, 0, uintptr(unsafe.Pointer(wide(text))))
		call("SendMessageW", n.historyList, 0x1a0, uintptr(i), n.px(n.historyRows[i].height))
	}
	for index, row := range rows {
		if selectedID != "" && row.id == selectedID {
			call("SendMessageW", n.historyList, 0x186, uintptr(index), 0)
			break
		}
	}
	call("SendMessageW", n.historyList, 0xb, 1, 0)
	visible(n.historyList, wasVisible && len(rows) > 0)
	call("InvalidateRect", n.historyList, 0, 1)
	if windowText(n.historyStatus) != status {
		call("SetWindowTextW", n.historyStatus, uintptr(unsafe.Pointer(wide(status))))
	}
	n.layoutHistory()
	if n.navigation.View != ViewHistory {
		n.navigate(ViewHistory)
	} else {
		n.layoutPreferences() // apply the new list/card height after expansion
	}
}

func (n *nativeDesktop) CurrentView() ViewState { return n.navigation.View }

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
	call("SendMessageW", n.historyRetention, 0x146, ^uintptr(0), uintptr(unsafe.Pointer(wide(label)))) // CB_SELECTSTRING over the whole list
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
		for _, h := range n.historyText {
			visible(h, false)
		}
	}
	if !show {
		return 0
	}
	count := len(n.historyRows)
	height := 0
	for _, row := range n.historyRows {
		height += row.height
	}
	if height > 276 {
		height = 276
	}
	if count == 0 {
		height = 88
	}
	visible(n.historyList, count > 0)
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

// updateHistoryButtons keeps one pool of per-row ghost icon buttons
// positioned over the list's actual item rectangles. All actions share one
// row, right-aligned with the device-name line.
func (n *nativeDesktop) updateHistoryButtons() {
	rows := n.historyRows
	need := len(rows) * historyKindsPerRow
	if n.historyListCallback == 0 {
		n.historyListCallback = syscall.NewCallback(func(h uintptr, msg uint32, w, l, id, ref uintptr) uintptr {
			result, _, _ := comctl.NewProc("DefSubclassProc").Call(h, uintptr(msg), w, l)
			if msg == 0x115 || msg == 0x20a || msg == 5 {
				n.positionHistoryButtons()
				n.positionHistoryText()
			}
			if msg == 0x201 || msg == 0x100 {
				n.positionHistoryButtons()
				n.positionHistoryText()
			}
			return result
		})
		comctl.NewProc("SetWindowSubclass").Call(n.historyList, n.historyListCallback, 3, 0)
	}
	if len(n.historyButtons) < need {
		previous := n.window
		n.window = n.home
		for len(n.historyButtons) < need {
			index := len(n.historyButtons)
			kind := index % historyKindsPerRow
			label, glyph := "Копировать", "\ue8c8"
			switch kind {
			case 1:
				label, glyph = "Показать значение", "\ue7b3"
			case 2:
				label, glyph = "Открыть ссылку", "\ue8a7"
			case 3:
				label, glyph = "Удалить запись", "\ue74d"
			case 4:
				label, glyph = "Развернуть запись", "\ue76c"
			}
			h := n.iconButton(label, glyph, 0, 0, 32, historyButtonBase+index)
			c := n.controls[h]
			c.kind = 5 // ghost: no frame, no fill until hover.
			n.controls[h] = c
			n.historyButtons = append(n.historyButtons, h)
		}
		n.window = previous
	}
	n.positionHistoryButtons()
	n.positionHistoryText()
}

// Expanded text retains the full payload in a native read-only, scrollable
// body. A list row itself cannot grow beyond the Win32 255-pixel limit.
func (n *nativeDesktop) positionHistoryText() {
	previous := n.window
	n.window = n.home
	defer func() { n.window = previous }()
	for len(n.historyText) < len(n.historyRows) {
		h := n.control("EDIT", "", 0x00200844, 0, 0, 1, 1, 9700+len(n.historyText))
		c := n.controls[h]
		c.kind = 7 // borderless history body, not a settings input.
		n.controls[h] = c
		call("SetWindowPos", h, 0, 0, 0, 0, 0, 0x37)
		n.historyText = append(n.historyText, h)
	}
	var client winRect
	call("GetClientRect", n.historyList, uintptr(unsafe.Pointer(&client)))
	for i, h := range n.historyText {
		if n.navigation.View != ViewHistory || i >= len(n.historyRows) || !n.historyRows[i].expanded {
			visible(h, false)
			if i >= len(n.historyRows) || !n.historyRows[i].expanded {
				call("SetWindowTextW", h, uintptr(unsafe.Pointer(wide(""))))
			}
			continue
		}
		var r winRect
		call("SendMessageW", n.historyList, 0x198, uintptr(i), uintptr(unsafe.Pointer(&r)))
		r.Top += int32(n.px(26))
		r.Bottom -= int32(n.px(4))
		if r.Top < 0 || r.Bottom > client.Bottom || r.Bottom <= r.Top {
			visible(h, false)
			continue
		}
		point := struct{ X, Y int32 }{int32(n.px(16)), r.Top}
		call("SendMessageW", h, 0x30, n.fontFor(n.home, 0), 0)
		call("MapWindowPoints", n.historyList, n.home, uintptr(unsafe.Pointer(&point)), 1)
		call("SetWindowPos", h, 0, uintptr(point.X), uintptr(point.Y), uintptr(client.Right-int32(n.px(24))), uintptr(r.Bottom-r.Top), 0x10)
		if windowText(h) != n.historyRows[i].preview {
			call("SetWindowTextW", h, uintptr(unsafe.Pointer(wide(n.historyRows[i].preview))))
		}
		visible(h, true)
	}
}

func (n *nativeDesktop) positionHistoryButtons() {
	var client winRect
	call("GetClientRect", n.historyList, uintptr(unsafe.Pointer(&client)))
	for index, h := range n.historyButtons {
		row, kind := index/historyKindsPerRow, index%historyKindsPerRow
		if n.navigation.View != ViewHistory || row >= len(n.historyRows) {
			visible(h, false)
			continue
		}
		row0 := n.historyRows[row]
		// Only visible actions occupy space: no detached expand button.
		order := []int{3, 0, 1, 2, 4}
		slot := 0
		for _, other := range order {
			if other == kind {
				break
			}
			if other == 1 && !row0.sensitive || other == 2 && !row0.url || other == 4 && !row0.truncated {
				continue
			}
			slot++
		}
		switch {
		case row0.group:
			visible(h, false)
			continue
		case kind == 1 && !row0.sensitive, kind == 2 && !row0.url, kind == 4 && !row0.truncated:
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
		x := client.Right - size - 8 - int32(slot)*int32(float64(38)*n.scale)
		// Buttons align with the device-name header line, not the row center.
		y := r.Top + int32(float64(17)*n.scale) - size/2
		point := struct{ X, Y int32 }{x, y}
		call("MapWindowPoints", n.historyList, n.home, uintptr(unsafe.Pointer(&point)), 1)
		// MapWindowPoints already includes the scrolled list's actual position.
		call("SetWindowPos", h, 0, uintptr(point.X), uintptr(point.Y), uintptr(size), uintptr(size), 0x10)
		font := n.fontFor(n.home, 5)
		if call("SendMessageW", h, 0x31, 0, 0) != font {
			call("SendMessageW", h, 0x30, font, 0)
		}
		glyph := "\ue8c8"
		switch kind {
		case 1:
			glyph = "\ue7b3"
			if row0.revealed {
				glyph = "eye-off" // drawn as a vector; no missing-font square.
			}
		case 2:
			glyph = "\ue8a7"
		case 3:
			glyph = "\ue74d"
		case 4:
			glyph = "\ue76c"
			if row0.expanded {
				glyph = "\ue70d"
			}
		}
		if n.icons[h] != glyph {
			n.icons[h] = glyph
			call("InvalidateRect", h, 0, 1)
		}
		call("InvalidateRect", h, 0, 0) // selection colors follow the underlying row
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
	case 4:
		if target.truncated {
			n.events = append(n.events, eventHistoryExpand)
		}
	}
}
