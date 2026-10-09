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
		h := call("CreateWindowExW", 0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(wide(title))), 0x42000000,
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

func (n *nativeDesktop) drawItem(d drawItem) {
	c := n.controls[d.Window]
	dpi := int(call("GetDpiForWindow", c.parent))
	if dpi == 0 {
		dpi = 96
	}
	px := func(v int) int32 { return int32(v * dpi / 96) }
	t := n.theme
	if d.Window == n.statusDot {
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
		fill, fg := t.surface, t.text
		if d.State&1 != 0 {
			fill = blendColor(t.surface, t.accent, .14)
		}
		if t.contrast && d.State&1 != 0 {
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
				n.drawText(d.DC, n.fontFor(c.parent, 0), syscall.UTF16ToString(text), r, fg, 0x8024)
			}
		}
		return
	}
	if d.Type == 4 { // native owner-drawn button
		fill, border, fg := t.surface, t.border, t.text
		if c.kind == 1 || c.kind == 3 {
			fill, border, fg = t.accent, t.accent, t.onAccent
		}
		if c.kind == 2 {
			fg = t.danger
		}
		if d.State&4 != 0 {
			fill, border, fg = t.surface, t.border, t.muted
		}
		if d.State&1 != 0 {
			fill = blendColor(fill, t.text, .1)
		}
		r := d.Rect
		r.Left++
		r.Top++
		r.Right--
		r.Bottom--
		radius, role := int(px(6)), 0
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
		if c.kind == 4 || d.Window == n.advancedButton {
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
		if d.State&0x10 != 0 {
			r.Left += px(4)
			r.Top += px(4)
			r.Right -= px(4)
			r.Bottom -= px(4)
			call("DrawFocusRect", d.DC, uintptr(unsafe.Pointer(&r)))
		}
		return
	}
	if d.Type != 2 {
		return
	}
	selected := d.State&1 != 0
	fill := t.surface
	if selected {
		fill = blendColor(t.surface, t.accent, .13)
	}
	if t.contrast && selected {
		fill = t.accent
	}
	brush := gcall("CreateSolidBrush", uintptr(fill))
	call("FillRect", d.DC, uintptr(unsafe.Pointer(&d.Rect)), brush)
	gcall("DeleteObject", brush)
	rows := n.rows[d.Window]
	if d.Item == 0xffffffff || int(d.Item) >= len(rows) {
		return
	}
	row := rows[d.Item]
	fg, muted := t.text, t.muted
	if t.contrast && selected {
		fg, muted = t.onAccent, t.onAccent
	}
	r := d.Rect
	r.Left += px(16)
	r.Right -= px(12)
	if d.Window == n.homeList {
		r.Right -= px(40)
		dot := t.muted
		if row.online {
			dot = t.online
		}
		if t.contrast && selected {
			dot = fg
		}
		n.roundRect(d.DC, winRect{r.Left, r.Top + px(22), r.Left + px(7), r.Top + px(29)}, dot, dot, int(px(4)))
		r.Left += px(20)
	}
	nameRect := r
	nameRect.Top += px(8)
	nameRect.Bottom = nameRect.Top + px(23)
	n.drawText(d.DC, n.fontFor(c.parent, 0), row.name, nameRect, fg, 0x8020)
	r.Top += px(32)
	r.Bottom = r.Top + px(20)
	n.drawText(d.DC, n.fontFor(c.parent, 4), row.detail, r, muted, 0x8020)
	if d.State&0x10 != 0 {
		r = d.Rect
		r.Left += 2
		r.Right -= 2
		call("DrawFocusRect", d.DC, uintptr(unsafe.Pointer(&r)))
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
	call("SendMessageW", hwnd, 0xb, 0, 0)
	n.rows[hwnd] = rows
	call("SendMessageW", hwnd, 0x184, 0, 0)
	for _, row := range rows {
		call("SendMessageW", hwnd, 0x180, 0, uintptr(unsafe.Pointer(wide(row.name+" — "+row.detail))))
	}
	if selection >= 0 && selection < len(rows) {
		call("SendMessageW", hwnd, 0x186, uintptr(selection), 0)
	}
	call("SendMessageW", hwnd, 0xb, 1, 0)
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
	for h, c := range n.controls {
		if c.parent != hwnd {
			continue
		}
		r := n.scaledRect(hwnd, c.bounds)
		r.Left -= window.scrollX
		r.Right -= window.scrollX
		r.Top -= window.scrollY
		r.Bottom -= window.scrollY
		call("SetWindowPos", h, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x14)
		call("SendMessageW", h, 0x30, n.fontFor(hwnd, c.role), 1)
		if c.class == "COMBOBOX" {
			call("SendMessageW", h, 0x153, ^uintptr(0), uintptr(n.scaledRect(hwnd, logicalRect{h: 26}).Bottom))
			call("SendMessageW", h, 0x153, 0, uintptr(n.scaledRect(hwnd, logicalRect{h: 32}).Bottom))
		}
		if h == n.homeList || h == n.foundList {
			call("SendMessageW", h, 0x1a0, 0, uintptr(n.scaledRect(hwnd, logicalRect{h: 60}).Bottom))
		}
	}
	if hwnd == n.home {
		n.updateRowTrash()
	}
	call("RedrawWindow", hwnd, 0, 0, 0x81)
}
