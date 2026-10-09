//go:build windows

package ui

import "unsafe"

func (n *nativeDesktop) drawPlatform(dc uintptr, r winRect, platform string, color uint32) {
	brush := gcall("CreateSolidBrush", uintptr(color))
	oldBrush := gcall("SelectObject", dc, brush)
	defer func() { gcall("SelectObject", dc, oldBrush); gcall("DeleteObject", brush) }()
	x := func(v int) int32 { return r.Left + int32(v)*(r.Right-r.Left)/20 }
	y := func(v int) int32 { return r.Top + int32(v)*(r.Bottom-r.Top)/20 }
	if platform == "windows" {
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				rect := winRect{x(2 + i*9), y(2 + j*9), x(9 + i*9), y(9 + j*9)}
				call("FillRect", dc, uintptr(unsafe.Pointer(&rect)), brush)
			}
		}
		return
	}
	if platform == "darwin" {
		type point struct{ X, Y int32 }
		curve := func(values ...int) {
			points := make([]point, len(values)/2)
			for i := range points {
				points[i] = point{x(values[i*2]), y(values[i*2+1])}
			}
			gcall("PolyBezierTo", dc, uintptr(unsafe.Pointer(&points[0])), uintptr(len(points)))
		}
		gcall("BeginPath", dc)
		gcall("MoveToEx", dc, uintptr(x(10)), uintptr(y(6)), 0)
		curve(5, 2, 2, 7, 3, 12, 4, 17, 6, 19, 8, 18, 10, 16, 11, 17, 13, 18, 15, 19, 18, 14, 18, 12, 14, 11, 14, 7, 17, 6, 14, 3, 12, 4, 10, 6)
		gcall("CloseFigure", dc)
		gcall("MoveToEx", dc, uintptr(x(10)), uintptr(y(4)), 0)
		curve(10, 1, 13, 0, 15, 0, 15, 3, 12, 5, 10, 4)
		gcall("CloseFigure", dc)
		gcall("EndPath", dc)
		gcall("FillPath", dc)
		return
	}
	// Unknown platforms use a neutral monitor, never a guessed brand.
	pen := gcall("CreatePen", 0, uintptr(n.px(1)), uintptr(color))
	oldPen := gcall("SelectObject", dc, pen)
	gcall("SelectObject", dc, gcall("GetStockObject", 5))
	gcall("RoundRect", dc, uintptr(x(1)), uintptr(y(2)), uintptr(x(19)), uintptr(y(15)), uintptr(n.px(2)), uintptr(n.px(2)))
	gcall("MoveToEx", dc, uintptr(x(10)), uintptr(y(15)), 0)
	gcall("LineTo", dc, uintptr(x(10)), uintptr(y(18)))
	gcall("MoveToEx", dc, uintptr(x(5)), uintptr(y(18)), 0)
	gcall("LineTo", dc, uintptr(x(15)), uintptr(y(18)))
	gcall("SelectObject", dc, oldPen)
	gcall("DeleteObject", pen)
}

func (n *nativeDesktop) iconButton(label, glyph string, x, y, w, id int) uintptr {
	h := n.control("BUTTON", label, 0x10000, x, y, w, 32, id)
	n.icons[h] = glyph
	if n.tooltip == 0 {
		n.tooltip = call("CreateWindowExW", 8, uintptr(unsafe.Pointer(wide("tooltips_class32"))), 0, 0x80000003, 0, 0, 0, 0, n.main, 0, n.instance, 0)
		call("SendMessageW", n.tooltip, 0x418, 0, n.px(360)) // TTM_SETMAXTIPWIDTH: wrap longer explanations.
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
