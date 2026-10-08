//go:build windows

package ui

import "unsafe"

func (n *nativeDesktop) iconButton(label, glyph string, x, y, w, id int) uintptr {
	h := n.control("BUTTON", label, 0x10000, x, y, w, 32, id)
	n.icons[h] = glyph
	if n.tooltip == 0 {
		n.tooltip = call("CreateWindowExW", 8, uintptr(unsafe.Pointer(wide("tooltips_class32"))), 0, 0x80000003, 0, 0, 0, 0, n.main, 0, n.instance, 0)
	}
	text := wide(label)
	n.tooltipText = append(n.tooltipText, text)
	var info struct {
		Size, Flags                     uint32
		Window, ID                      uintptr
		Rect                            winRect
		Instance, Text, Param, Reserved uintptr
	}
	info.Size = uint32(unsafe.Sizeof(info))
	info.Flags = 0x11
	info.Window = n.window
	info.ID = h
	info.Text = uintptr(unsafe.Pointer(text))
	call("SendMessageW", n.tooltip, 0x432, 0, uintptr(unsafe.Pointer(&info)))
	return h
}

func (n *nativeDesktop) drawPlus(dc uintptr, r winRect, fg uint32, width int32) {
	x, y := (r.Left+r.Right)/2, (r.Top+r.Bottom)/2
	arm := (r.Right - r.Left) / 7
	pen := gcall("CreatePen", 0, uintptr(width), uintptr(fg))
	old := gcall("SelectObject", dc, pen)
	gcall("MoveToEx", dc, uintptr(x-arm), uintptr(y), 0)
	gcall("LineTo", dc, uintptr(x+arm+1), uintptr(y))
	gcall("MoveToEx", dc, uintptr(x), uintptr(y-arm), 0)
	gcall("LineTo", dc, uintptr(x), uintptr(y+arm+1))
	gcall("SelectObject", dc, old)
	gcall("DeleteObject", pen)
}
