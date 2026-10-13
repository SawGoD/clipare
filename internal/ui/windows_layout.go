//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

type winRect struct{ Left, Top, Right, Bottom int32 }
type logicalRect struct{ x, y, w, h int }
type winControl struct {
	parent     uintptr
	class      string
	bounds     logicalRect
	role, kind int
}
type winWindow struct {
	width, height    int
	cards            []logicalRect
	scrollX, scrollY int32
}
type drawItem struct {
	Type, ID, Item, Action, State uint32
	Window, DC                    uintptr
	Rect                          winRect
	Data                          uintptr
}

func (n *nativeDesktop) scaledRect(hwnd uintptr, r logicalRect) winRect {
	dpi := int(call("GetDpiForWindow", hwnd))
	if dpi == 0 {
		dpi = 96
	}
	return winRect{int32(r.x * dpi / 96), int32(r.y * dpi / 96), int32((r.x + r.w) * dpi / 96), int32((r.y + r.h) * dpi / 96)}
}

func (n *nativeDesktop) panel(class *uint16, title string, width, height int, cards ...logicalRect) uintptr {
	if n.main != 0 {
		h := call("CreateWindowExW", 0x02000000, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(wide(title))), 0x42000000,
			0, 0, n.px(width), n.px(height), n.main, 0, n.instance, 0)
		n.windows[h] = winWindow{width: width, height: height, cards: cards}
		return h
	}
	r := winRect{Right: int32(n.px(width)), Bottom: int32(n.px(height))}
	style := uintptr(0x02CA0000) // native caption, clip children during background painting
	call("AdjustWindowRectEx", uintptr(unsafe.Pointer(&r)), style, 0, 0)
	h := call("CreateWindowExW", 0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(wide(title))), style,
		0x80000000, 0x80000000, uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0, 0, n.instance, 0)
	n.windows[h] = winWindow{width: width, height: height, cards: cards}
	n.fitWindow(h)
	return h
}

func (n *nativeDesktop) heading(text string, x, y, w int, role int) uintptr {
	h := n.control("STATIC", text, 0x4000, x, y, w, 30, 0) // end ellipsis
	n.setRole(h, role)
	return h
}
func (n *nativeDesktop) setRole(h uintptr, role int) {
	c := n.controls[h]
	c.role = role
	n.controls[h] = c
	call("SendMessageW", h, 0x30, n.fontFor(c.parent, role), 1)
}

// Surface painting uses GDI only; controls retain their HWND, keyboard and accessibility.
func (n *nativeDesktop) roundRect(dc uintptr, r winRect, fill, border uint32, radius int) {
	brush := gcall("CreateSolidBrush", uintptr(fill))
	pen := gcall("CreatePen", 0, 1, uintptr(border))
	oldBrush, oldPen := gcall("SelectObject", dc, brush), gcall("SelectObject", dc, pen)
	gcall("RoundRect", dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(radius*2), uintptr(radius*2))
	gcall("SelectObject", dc, oldBrush)
	gcall("SelectObject", dc, oldPen)
	gcall("DeleteObject", brush)
	gcall("DeleteObject", pen)
}

func (n *nativeDesktop) drawText(dc, font uintptr, text string, r winRect, fg uint32, flags uintptr) {
	old := gcall("SelectObject", dc, font)
	gcall("SetTextColor", dc, uintptr(fg))
	gcall("SetBkMode", dc, 1)
	call("DrawTextW", dc, uintptr(unsafe.Pointer(wide(text))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), flags|0x800) // no prefix
	gcall("SelectObject", dc, old)
}

func (n *nativeDesktop) paintRow(dc uintptr, r winRect, fill uint32, px func(int) int32) {
	call("FillRect", dc, uintptr(unsafe.Pointer(&r)), n.surfaceBrush)
	if fill == n.theme.surface {
		return
	}
	if !n.theme.contrast {
		r.Left += px(4)
		r.Right -= px(4)
		r.Bottom -= px(2)
	}
	n.roundRect(dc, r, fill, fill, int(px(fluentControlRadius)))
}

func (n *nativeDesktop) drawItem(d drawItem) {
	c := n.controls[d.Window]
	dpi := int(call("GetDpiForWindow", c.parent))
	if dpi == 0 {
		dpi = 96
	}
	px := func(v int) int32 { return int32(v * dpi / 96) }
	t := n.theme
	if d.Window == n.statusDot {
		call("FillRect", d.DC, uintptr(unsafe.Pointer(&d.Rect)), n.surfaceBrush)
		r := d.Rect
		r.Top += px(6)
		r.Bottom = r.Top + px(10)
		r.Right = r.Left + px(10)
		fg := color(n.utilityStatus.Color())
		if t.dark && n.utilityStatus.State == SyncActive {
			fg = t.online
		}
		if t.contrast {
			fg = t.text
		}
		n.roundRect(d.DC, r, fg, fg, int(px(5)))
		return
	}
	if d.Type == 3 {
		fill, fg := t.inputFill(), t.text
		if d.State&1 != 0 && d.State&0x1000 == 0 { // ODS_COMBOBOXEDIT: never highlight closed value
			fill = blendColor(t.surface, t.accent, .14)
		}
		if t.contrast && d.State&1 != 0 && d.State&0x1000 == 0 {
			fill, fg = t.accent, t.onAccent
		}
		brush := gcall("CreateSolidBrush", uintptr(fill))
		call("FillRect", d.DC, uintptr(unsafe.Pointer(&d.Rect)), brush)
		gcall("DeleteObject", brush)
		if d.Item != ^uint32(0) {
			length := call("SendMessageW", d.Window, 0x149, uintptr(d.Item), 0)
			if length < 8192 {
				text := make([]uint16, length+1)
				call("SendMessageW", d.Window, 0x148, uintptr(d.Item), uintptr(unsafe.Pointer(&text[0])))
				r := d.Rect
				r.Left += px(10)
				r.Right -= px(10)
				r.Top += px(2)
				n.drawText(d.DC, n.fontFor(c.parent, 0), syscall.UTF16ToString(text), r, fg, 0x8024)
			}
		}
		return
	}
	if d.Type == 4 { // native owner-drawn button
		base := t.surface
		for _, card := range n.windows[c.parent].cards {
			if c.bounds.x >= card.x && c.bounds.y >= card.y && c.bounds.y < card.y+card.h {
				base = t.surface
				break
			}
			base = t.background
		}
		fill, border, fg := t.buttonColors(c.kind, d.State&4 != 0, n.isHovered(d.Window), d.State&1 != 0, base)
		if c.kind == 5 || c.kind == 6 { // ghost row actions: card-colored, no frame.
			if id := int(call("GetDlgCtrlID", d.Window)); id >= historyButtonBase && id < historyButtonBase+len(n.historyButtons) {
				selected := int(int32(call("SendMessageW", n.historyList, 0x188, 0, 0)))
				base = n.rowBackground(n.historyList, (id-historyButtonBase)/historyKindsPerRow, (id-historyButtonBase)/historyKindsPerRow == selected)
				if (id-historyButtonBase)/historyKindsPerRow == selected {
					if t.contrast {
						fg = t.onAccent
					}
				}
			} else if id >= 9000 && id < 9000+len(n.rowTrash) {
				selected := int(int32(call("SendMessageW", n.homeList, 0x188, 0, 0)))
				base = n.rowBackground(n.homeList, id-9000, id-9000 == selected)
			}
			if t.contrast && base == t.accent {
				fg = t.onAccent
			}
			brush := gcall("CreateSolidBrush", uintptr(base))
			call("FillRect", d.DC, uintptr(unsafe.Pointer(&d.Rect)), brush)
			gcall("DeleteObject", brush)
			fill, _, actionFG := t.buttonColors(c.kind, d.State&4 != 0, n.isHovered(d.Window), d.State&1 != 0, base)
			if !(t.contrast && base == t.accent) {
				fg = actionFG
			}
			if fill != base {
				n.roundRect(d.DC, d.Rect, fill, fill, int(px(fluentControlRadius)))
			}
			label := n.icons[d.Window]
			if label == "" {
				label = windowText(d.Window)
			}
			if label == "eye-off" {
				n.drawHiddenEye(d.DC, d.Rect, fg, px)
			} else {
				n.drawText(d.DC, n.fontFor(c.parent, 5), label, d.Rect, fg, 0x25)
			}
			if d.State&0x10 != 0 && n.keyboardFocusVisible() {
				call("DrawFocusRect", d.DC, uintptr(unsafe.Pointer(&d.Rect)))
			}
			return
		}
		r := d.Rect
		r.Left++
		r.Top++
		r.Right--
		r.Bottom--
		radius, role := int(px(fluentControlRadius)), 0
		if c.kind == 3 {
			radius, role = int(px(40)), 3
		}
		n.roundRect(d.DC, r, fill, border, radius)
		label := windowText(d.Window)
		if icon := n.icons[d.Window]; icon != "" {
			label, role = icon, 5
		}
		if c.kind == 3 {
			n.drawPlus(d.DC, r, fg, px(2))
			label = ""
		}
		if c.kind == 4 {
			labelRect := r
			labelRect.Left += px(12)
			labelRect.Right -= px(32)
			n.drawText(d.DC, n.fontFor(c.parent, role), label, labelRect, fg, 0x8024)
			arrow := "\uE76C" // Fluent ChevronRight.
			expanded := n.preferencesExpanded
			if d.Window == n.advancedButton {
				expanded = n.navigation.AdvancedExpanded
			}
			if expanded {
				arrow = "\uE70D" // Fluent ChevronDown.
			}
			arrowRect := r
			arrowRect.Left = arrowRect.Right - px(32)
			n.drawText(d.DC, n.fontFor(c.parent, 5), arrow, arrowRect, fg, 0x25)
		} else {
			n.drawText(d.DC, n.fontFor(c.parent, role), label, r, fg, 0x25|0x8000)
		}
		if d.State&0x10 != 0 && n.keyboardFocusVisible() {
			r.Left += px(4)
			r.Top += px(4)
			r.Right -= px(4)
			r.Bottom -= px(4)
			call("DrawFocusRect", d.DC, uintptr(unsafe.Pointer(&r)))
		}
		return
	}
	if d.Window == n.historyList && d.Type == 2 {
		n.drawHistoryItem(d, px)
		return
	}
	if d.Type != 2 {
		return
	}
	selected := d.State&1 != 0
	fill := n.rowBackground(d.Window, int(d.Item), selected)
	n.paintRow(d.DC, d.Rect, fill, px)
	rows := n.rows[d.Window]
	if d.Item == 0xffffffff || int(d.Item) >= len(rows) {
		return
	}
	row := rows[d.Item]
	if int(d.Item)+1 < len(rows) && !selected {
		line := winRect{d.Rect.Left + px(16), d.Rect.Bottom - px(1), d.Rect.Right - px(16), d.Rect.Bottom}
		brush := gcall("CreateSolidBrush", uintptr(blendColor(t.surface, t.border, .45)))
		call("FillRect", d.DC, uintptr(unsafe.Pointer(&line)), brush)
		gcall("DeleteObject", brush)
	}
	fg, muted := t.text, t.muted
	if t.contrast && selected {
		fg, muted = t.onAccent, t.onAccent
	}
	r := d.Rect
	r.Left += px(16)
	r.Right -= px(12)
	n.drawPlatform(d.DC, winRect{r.Left, r.Top + px(18), r.Left + px(20), r.Top + px(38)}, row.platform, muted)
	r.Left += px(32)
	if d.Window == n.homeList {
		r.Right -= px(40)
		dot := t.muted
		if row.online {
			dot = t.online
		}
		if t.contrast && selected {
			dot = fg
		}
		n.roundRect(d.DC, winRect{r.Left, r.Top + px(39), r.Left + px(6), r.Top + px(45)}, dot, dot, int(px(3)))
	}
	nameRect := r
	nameRect.Top += px(8)
	nameRect.Bottom = nameRect.Top + px(23)
	n.drawText(d.DC, n.fontFor(c.parent, 0), row.name, nameRect, fg, 0x8020)
	r.Top += px(32)
	r.Bottom = r.Top + px(20)
	if d.Window == n.homeList {
		r.Left += px(12)
	}
	n.drawText(d.DC, n.fontFor(c.parent, 4), row.detail, r, muted, 0x8020)
	if d.State&0x10 != 0 && n.keyboardFocusVisible() {
		r = d.Rect
		r.Left += 2
		r.Right -= 2
		call("DrawFocusRect", d.DC, uintptr(unsafe.Pointer(&r)))
	}
}

// drawHistoryItem renders one history row: group label, source header and a
// short preview (masked for hidden secrets, accent-colored for links).
func (n *nativeDesktop) drawHistoryItem(d drawItem, px func(int) int32) {
	t := n.theme
	selected := d.State&1 != 0
	fill := n.rowBackground(d.Window, int(d.Item), selected)
	n.paintRow(d.DC, d.Rect, fill, px)
	if d.Item == 0xffffffff || int(d.Item) >= len(n.historyRows) {
		return
	}
	row := n.historyRows[d.Item]
	fg, muted := t.text, t.muted
	if t.contrast && selected {
		fg, muted = t.onAccent, t.onAccent
	}
	r := d.Rect
	r.Left += px(16)
	r.Right -= px(8)
	if row.group {
		// Center the label within the compact group row deterministically.
		lineH := px(18)
		top := r.Top + ((r.Bottom-r.Top)-lineH)/2
		n.drawText(d.DC, n.fontFor(n.home, 4), row.header, winRect{r.Left, top, r.Right, top + lineH}, muted, 0x25)
		return
	}
	headerRect := r
	headerRect.Right -= px(196) // action strip only overlaps the header, never the body
	headerRect.Top += px(8)
	headerRect.Bottom = headerRect.Top + px(18)
	n.drawText(d.DC, n.fontFor(n.home, 4), row.header, headerRect, muted, 0x8020)
	previewColor := fg
	if row.url {
		previewColor = t.accent
	}
	previewRect := r
	previewRect.Top += px(26)
	previewRect.Bottom = d.Rect.Bottom - px(4)
	// Word wrap over two lines; DT_END_ELLIPSIS cuts the last line.
	if !row.expanded {
		n.drawText(d.DC, n.fontFor(n.home, 0), row.preview, previewRect, previewColor, 0x8010)
	}
	if d.State&0x10 != 0 && n.keyboardFocusVisible() {
		call("DrawFocusRect", d.DC, uintptr(unsafe.Pointer(&d.Rect)))
	}
}

func blendColor(a, b uint32, f float64) uint32 {
	var out uint32
	for s := uint(0); s < 24; s += 8 {
		out |= uint32(float64((a>>s)&255)*(1-f)+float64((b>>s)&255)*f) << s
	}
	return out
}

func (n *nativeDesktop) deviceList(x, y, w, h, id int) uintptr {
	hwnd := n.control("LISTBOX", "", 0x00210151, x, y, w, h, id) // strings, fixed owner draw, notify, scroll, tab
	call("SendMessageW", hwnd, 0x1a0, 0, uintptr(n.scaledRect(n.window, logicalRect{h: 60}).Bottom))
	return hwnd
}

func (n *nativeDesktop) setRows(hwnd uintptr, rows []deviceRow) {
	selection := int(int32(call("SendMessageW", hwnd, 0x188, 0, 0)))
	// WM_SETREDRAW(TRUE) restores WS_VISIBLE even for a hidden list.
	wasVisible := call("GetWindowLongPtrW", hwnd, ^uintptr(15))&0x10000000 != 0
	call("SendMessageW", hwnd, 0xb, 0, 0)
	n.rows[hwnd] = rows
	call("SendMessageW", hwnd, 0x184, 0, 0)
	for _, row := range rows {
		call("SendMessageW", hwnd, 0x180, 0, uintptr(unsafe.Pointer(wide(platformName(row.platform)+": "+row.name+" — "+row.detail))))
	}
	if selection >= 0 && selection < len(rows) {
		call("SendMessageW", hwnd, 0x186, uintptr(selection), 0)
	}
	call("SendMessageW", hwnd, 0xb, 1, 0)
	visible(hwnd, wasVisible)
	call("InvalidateRect", hwnd, 0, 1)
	if hwnd == n.homeList {
		n.updateRowTrash()
	}
}

func (n *nativeDesktop) dpiChanged(hwnd uintptr, w, l uintptr) {
	var r winRect
	k32.NewProc("RtlMoveMemory").Call(uintptr(unsafe.Pointer(&r)), l, unsafe.Sizeof(r))
	call("SetWindowPos", hwnd, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x14)
	n.fitWindow(hwnd)
	n.layoutControls(hwnd)
}

func (n *nativeDesktop) layoutControls(hwnd uintptr) {
	window := n.windows[hwnd]
	type move struct {
		h uintptr
		r winRect
	}
	var moves []move
	for h, c := range n.controls {
		if c.parent != hwnd {
			continue
		}
		if id := call("GetDlgCtrlID", h); id >= 9000 && id < 9000+uintptr(len(n.rowTrash)) {
			continue // Positioned from the list's actual item rectangles below.
		}
		if id := call("GetDlgCtrlID", h); id >= historyButtonBase && id < historyButtonBase+uintptr(len(n.historyButtons)) {
			continue
		}
		if c.kind == 7 {
			continue
		} // positioned inside its expanded history row
		r := n.scaledRect(hwnd, c.bounds)
		r.Left -= window.scrollX
		r.Right -= window.scrollX
		r.Top -= window.scrollY
		r.Bottom -= window.scrollY
		moves = append(moves, move{h, r})
		font := n.fontFor(hwnd, c.role)
		fontChanged := call("SendMessageW", h, 0x31, 0, 0) != font
		if fontChanged {
			call("SendMessageW", h, 0x30, font, 0)
		}
		if fontChanged && c.class == "COMBOBOX" {
			call("SendMessageW", h, 0x153, ^uintptr(0), uintptr(n.scaledRect(hwnd, logicalRect{h: 26}).Bottom))
			call("SendMessageW", h, 0x153, 0, uintptr(n.scaledRect(hwnd, logicalRect{h: 32}).Bottom))
		}
		if fontChanged && (h == n.homeList || h == n.foundList) {
			call("SendMessageW", h, 0x1a0, 0, uintptr(n.scaledRect(hwnd, logicalRect{h: 60}).Bottom))
		}
	}
	// Commit positions together, without intermediate erase/font redraws.
	batch := call("BeginDeferWindowPos", uintptr(len(moves)))
	for _, m := range moves {
		if batch == 0 {
			break
		}
		r := m.r
		batch = call("DeferWindowPos", batch, m.h, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x1c)
	}
	if batch == 0 || call("EndDeferWindowPos", batch) == 0 {
		for _, m := range moves {
			r := m.r
			call("SetWindowPos", m.h, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x1c)
		}
	}
	if hwnd == n.home {
		n.updateRowTrash()
		if len(n.historyButtons) > 0 || n.navigation.View == ViewHistory {
			n.updateHistoryButtons()
		}
	}
	// Card geometry is drawn in WM_ERASEBKGND. Invalidation alone leaves
	// obsolete card edges and stripes behind moved controls.
	call("RedrawWindow", hwnd, 0, 0, 0x85)
}
