//go:build windows

package ui

import "unsafe"

type scrollInfo struct {
	Size, Mask uint32
	Min, Max   int32
	Page       uint32
	Pos, Track int32
}

func (n *nativeDesktop) fitWindow(hwnd uintptr) {
	if _, ok := n.windows[hwnd]; !ok {
		return
	}
	var monitor struct {
		Size          uint32
		Monitor, Work winRect
		Flags         uint32
	}
	monitor.Size = uint32(unsafe.Sizeof(monitor))
	h := call("MonitorFromWindow", hwnd, 2)
	if call("GetMonitorInfoW", h, uintptr(unsafe.Pointer(&monitor))) == 0 {
		return
	}
	var r winRect
	call("GetWindowRect", hwnd, uintptr(unsafe.Pointer(&r)))
	w, hgt := r.Right-r.Left, r.Bottom-r.Top
	if max := monitor.Work.Right - monitor.Work.Left - 32; w > max {
		w = max
	}
	if max := monitor.Work.Bottom - monitor.Work.Top - 32; hgt > max {
		hgt = max
	}
	x, y := r.Left, r.Top
	if x+w > monitor.Work.Right {
		x = monitor.Work.Right - w - 16
	}
	if x < monitor.Work.Left {
		x = monitor.Work.Left + 16
	}
	if y+hgt > monitor.Work.Bottom {
		y = monitor.Work.Bottom - hgt - 16
	}
	if y < monitor.Work.Top {
		y = monitor.Work.Top + 16
	}
	call("SetWindowPos", hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(hgt), 0x14)
	n.updateScroll(hwnd)
}

func (n *nativeDesktop) updateScroll(hwnd uintptr) {
	if n.layingOut {
		return
	}
	window, ok := n.windows[hwnd]
	if !ok {
		return
	}
	n.layingOut = true
	defer func() { n.layingOut = false }()
	var client winRect
	call("GetClientRect", hwnd, uintptr(unsafe.Pointer(&client)))
	content := n.scaledRect(hwnd, logicalRect{w: window.width, h: window.height})
	for axis := uintptr(0); axis < 2; axis++ {
		extent, page, pos := content.Right, client.Right, window.scrollX
		if axis == 1 {
			extent, page, pos = content.Bottom, client.Bottom, window.scrollY
		}
		show := uintptr(0)
		if extent > page {
			show = 1
		}
		call("ShowScrollBar", hwnd, axis, show)
		call("GetClientRect", hwnd, uintptr(unsafe.Pointer(&client)))
		if axis == 0 {
			page = client.Right
		} else {
			page = client.Bottom
		}
		pos = clampScroll(pos, extent, page)
		info := scrollInfo{Size: uint32(unsafe.Sizeof(scrollInfo{})), Mask: 7, Max: extent - 1, Page: uint32(page), Pos: pos}
		call("SetScrollInfo", hwnd, axis, uintptr(unsafe.Pointer(&info)), 1)
		if axis == 0 {
			window.scrollX = pos
		} else {
			window.scrollY = pos
		}
	}
	n.windows[hwnd] = window
}

func (n *nativeDesktop) scrollWindow(hwnd uintptr, axis uintptr, delta int32, absolute bool) {
	info := scrollInfo{Size: uint32(unsafe.Sizeof(scrollInfo{})), Mask: 0x17}
	if call("GetScrollInfo", hwnd, axis, uintptr(unsafe.Pointer(&info))) == 0 {
		return
	}
	pos := info.Pos + delta
	if absolute {
		pos = delta
	}
	pos = clampScroll(pos, info.Max+1, int32(info.Page))
	window := n.windows[hwnd]
	if axis == 0 {
		window.scrollX = pos
	} else {
		window.scrollY = pos
	}
	n.windows[hwnd] = window
	info.Pos = pos
	info.Mask = 4
	call("SetScrollInfo", hwnd, axis, uintptr(unsafe.Pointer(&info)), 1)
	n.layoutControls(hwnd)
}

func (n *nativeDesktop) scrollMessage(hwnd uintptr, msg uint32, w, l uintptr) bool {
	if _, ok := n.windows[hwnd]; !ok {
		return false
	}
	switch msg {
	case 5:
		n.updateScroll(hwnd)
		n.layoutControls(hwnd)
		return true
	case 0x115, 0x114:
		axis := uintptr(1)
		if msg == 0x114 {
			axis = 0
		}
		info := scrollInfo{Size: uint32(unsafe.Sizeof(scrollInfo{})), Mask: 0x17}
		call("GetScrollInfo", hwnd, axis, uintptr(unsafe.Pointer(&info)))
		delta := int32(0)
		absolute := false
		switch w & 0xffff {
		case 0:
			delta = -32
		case 1:
			delta = 32
		case 2:
			delta = -int32(info.Page)
		case 3:
			delta = int32(info.Page)
		case 4, 5:
			delta = info.Track
			absolute = true
		case 6:
			delta = 0
			absolute = true
		case 7:
			delta = info.Max
			absolute = true
		default:
			return true
		}
		n.scrollWindow(hwnd, axis, delta, absolute)
		return true
	case 0x20a:
		n.scrollWindow(hwnd, 1, -int32(int16(w>>16))*48/120, false)
		return true
	}
	return false
}

func (n *nativeDesktop) revealFocus() {
	focus := call("GetFocus")
	if focus == n.lastFocus {
		return
	}
	n.lastFocus = focus
	c, ok := n.controls[focus]
	if !ok {
		return
	}
	window := n.windows[c.parent]
	r := n.scaledRect(c.parent, c.bounds)
	var client winRect
	call("GetClientRect", c.parent, uintptr(unsafe.Pointer(&client)))
	x, y := window.scrollX, window.scrollY
	if r.Right > x+client.Right {
		x = r.Right - client.Right
	}
	if r.Left < x {
		x = r.Left
	}
	if r.Bottom > y+client.Bottom {
		y = r.Bottom - client.Bottom
	}
	if r.Top < y {
		y = r.Top
	}
	if x != window.scrollX {
		n.scrollWindow(c.parent, 0, x, true)
	}
	if y != window.scrollY {
		n.scrollWindow(c.parent, 1, y, true)
	}
}
