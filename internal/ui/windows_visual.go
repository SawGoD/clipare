//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

// Logical pixels, converted at the destination HWND's current DPI.
// Win32 RoundRect takes a diameter; roundRect takes a radius.
const (
	fluentCardRadius    = 10
	fluentControlRadius = 5
	fluentHoverMix      = .045
	fluentPressedMix    = .08
	fluentSelectionMix  = .13
)

func (t winTheme) rowFill(selected, hot bool) uint32 {
	if selected {
		if t.contrast {
			return t.accent
		}
		return blendColor(t.surface, t.accent, fluentSelectionMix)
	}
	if hot && !t.contrast {
		return blendColor(t.surface, t.text, fluentHoverMix)
	}
	return t.surface
}

func (t winTheme) buttonColors(kind int, disabled, hot, pressed bool, base uint32) (fill, border, fg uint32) {
	fill, border, fg = base, t.border, t.text
	if kind == 1 || kind == 3 {
		fill, border, fg = t.accent, t.accent, t.onAccent
	}
	if kind == 2 || kind == 6 && (hot || pressed) {
		fg = t.danger
	}
	if kind == 6 && !hot && !pressed {
		fg = t.muted
	}
	if kind == 4 || kind == 5 || kind == 6 {
		border = fill
	}
	if disabled {
		return base, t.border, t.muted
	}
	if !t.contrast {
		if pressed {
			fill = blendColor(fill, fg, fluentPressedMix)
		} else if hot {
			fill = blendColor(fill, fg, fluentHoverMix)
		}
		if kind == 1 || kind == 3 || kind == 4 || kind == 5 || kind == 6 {
			border = fill
		}
	}
	return
}

// Owner-drawn BUTTON does not reliably supply ODS_HOT. Track hover without
// polling, animations or swallowing native input/accessibility messages.
func (n *nativeDesktop) trackVisualState(h uintptr) {
	if n.hovered == nil {
		n.hovered = make(map[uintptr]int)
	}
	if n.visualCallback == 0 {
		n.visualCallback = syscall.NewCallback(func(h uintptr, msg uint32, w, l, id, ref uintptr) uintptr {
			switch msg {
			case 0x200: // WM_MOUSEMOVE
				var tracking struct {
					Size, Flags uint32
					Window      uintptr
					Time        uint32
				}
				tracking.Size = uint32(unsafe.Sizeof(tracking))
				tracking.Flags, tracking.Window = 2, h // TME_LEAVE
				call("TrackMouseEvent", uintptr(unsafe.Pointer(&tracking)))
				if n.controls[h].class != "LISTBOX" {
					n.setHovered(h, 0, true)
				}
				n.refreshHoveredRows()
			case 0x2a3: // WM_MOUSELEAVE
				if n.controls[h].class != "LISTBOX" {
					n.setHovered(h, 0, false)
				}
				n.refreshHoveredRows()
			case 7, 8, 0xa, 0x201, 0x202: // focus, enabled, pressed/released
				if msg == 0xa && w == 0 {
					n.setHovered(h, 0, false)
				}
				call("InvalidateRect", h, 0, 0)
			case 0x82:
				delete(n.hovered, h)
				comctl.NewProc("RemoveWindowSubclass").Call(h, n.visualCallback, 8)
			}
			r, _, _ := comctl.NewProc("DefSubclassProc").Call(h, uintptr(msg), w, l)
			if msg == 0x115 || msg == 0x20a {
				n.refreshHoveredRows()
			}
			return r
		})
	}
	comctl.NewProc("SetWindowSubclass").Call(h, n.visualCallback, 8, 0)
}

func (n *nativeDesktop) setHovered(h uintptr, row int, hot bool) bool {
	previous, exists := n.hovered[h]
	if exists == hot && (!hot || previous == row) {
		return false
	}
	if hot {
		n.hovered[h] = row
	} else {
		delete(n.hovered, h)
	}
	call("InvalidateRect", h, 0, 0)
	return true
}

func (n *nativeDesktop) isHovered(h uintptr) bool {
	_, hot := n.hovered[h]
	return hot
}

func (n *nativeDesktop) refreshHoveredRows() {
	// Screen hit-testing includes action HWNDs above the list; their backgrounds
	// must match the row, not leave square patches on hover.
	changed := false
	for _, list := range []uintptr{n.homeList, n.foundList, n.historyList} {
		if list == 0 {
			continue
		}
		var p struct{ X, Y int32 }
		call("GetCursorPos", uintptr(unsafe.Pointer(&p)))
		call("ScreenToClient", list, uintptr(unsafe.Pointer(&p)))
		var r winRect
		call("GetClientRect", list, uintptr(unsafe.Pointer(&r)))
		hot := call("IsWindowVisible", list) != 0 && p.X >= 0 && p.Y >= 0 && p.X < r.Right && p.Y < r.Bottom
		item := call("SendMessageW", list, 0x1a9, uintptr(uint16(p.X))|uintptr(uint16(p.Y))<<16, 0) // LB_ITEMFROMPOINT
		hot = hot && item>>16 == 0
		var itemRect winRect
		if int32(call("SendMessageW", list, 0x198, item&0xffff, uintptr(unsafe.Pointer(&itemRect)))) < 0 || p.Y < itemRect.Top || p.Y >= itemRect.Bottom {
			hot = false // nearest item is not necessarily beneath the pointer
		}
		changed = n.setHovered(list, int(item&0xffff), hot) || changed
	}
	if !changed {
		return
	}
	for _, h := range append(append([]uintptr(nil), n.rowTrash...), n.historyButtons...) {
		if call("IsWindowVisible", h) != 0 {
			call("InvalidateRect", h, 0, 0)
		}
	}
}

func (n *nativeDesktop) rowBackground(list uintptr, row int, selected bool) uint32 {
	hover, hot := n.hovered[list]
	// Expanded rows host a native EDIT with its own background brush.
	// Keep that surface uniform; selection is still shared with the body.
	if list == n.historyList && row >= 0 && row < len(n.historyRows) && (n.historyRows[row].expanded || n.historyRows[row].group) {
		hot = false
	}
	return n.theme.rowFill(selected, hot && hover == row)
}

func (t winTheme) inputFill() uint32 {
	if t.contrast {
		return t.surface
	}
	return blendColor(t.surface, t.text, .025)
}
