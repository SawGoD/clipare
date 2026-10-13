//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

var comctl = syscall.NewLazyDLL("comctl32.dll")

// Keep native editing, automatic check state, keyboard navigation and MSAA.
// Subclassing changes painting only; all input goes through DefSubclassProc.
func (n *nativeDesktop) styleControl(h uintptr) {
	if n.controlCallback == 0 {
		n.controlCallback = syscall.NewCallback(func(h uintptr, msg uint32, w, l, id, ref uintptr) uintptr {
			c := n.controls[h]
			check := c.class == "BUTTON"
			if c.class == "COMBOBOX" && (msg == 0x133 || msg == 0x134) {
				gcall("SetTextColor", w, uintptr(n.theme.text))
				gcall("SetBkColor", w, uintptr(n.theme.inputFill()))
				return n.inputBrush
			}
			if check && (msg == 0xf || msg == 0x318) {
				if msg == 0x318 {
					n.paintCheckbox(h, w)
				} else {
					var ps struct {
						DC              uintptr
						Erase           int32
						Rect            winRect
						Restore, Update int32
						Reserved        [32]byte
					}
					dc := call("BeginPaint", h, uintptr(unsafe.Pointer(&ps)))
					n.paintCheckbox(h, dc)
					call("EndPaint", h, uintptr(unsafe.Pointer(&ps)))
				}
				return 0
			}
			if msg == 0x82 { // WM_NCDESTROY: remove while the native HWND is valid.
				comctl.NewProc("RemoveWindowSubclass").Call(h, n.controlCallback, 1)
			}
			r, _, _ := comctl.NewProc("DefSubclassProc").Call(h, uintptr(msg), w, l)
			if msg == 0x83 && c.class == "EDIT" && c.kind != 7 { // WM_NCCALCSIZE
				// Reserve a small top inset without replacing native text editing.
				var rect winRect
				k32.NewProc("RtlMoveMemory").Call(uintptr(unsafe.Pointer(&rect)), l, unsafe.Sizeof(rect))
				dpi := call("GetDpiForWindow", h)
				if dpi == 0 {
					dpi = 96
				}
				rect.Top += int32(2 * dpi / 96)
				k32.NewProc("RtlMoveMemory").Call(l, uintptr(unsafe.Pointer(&rect)), unsafe.Sizeof(rect))
			}
			if !check && c.kind != 7 && (msg == 0xf || msg == 0x85 || msg == 0x317 || msg == 0x318) {
				dc := uintptr(0)
				if msg == 0x317 || msg == 0x318 {
					dc = w
				}
				n.paintInputFrame(h, c.class == "COMBOBOX", dc)
			}
			switch msg {
			case 7, 8, 0xf1, 0xf3, 0x100, 0x101, 0x201, 0x202, 0xa:
				call("InvalidateRect", h, 0, 0)
			}
			return r
		})
	}
	comctl.NewProc("SetWindowSubclass").Call(h, n.controlCallback, 1, 0)
	if n.controls[h].class == "EDIT" {
		call("SetWindowPos", h, 0, 0, 0, 0, 0, 0x37) // recalculate native text inset
	}
}

// Draw a crossed-out eye with GDI paths, independent of the installed icon font.
func (n *nativeDesktop) drawHiddenEye(dc uintptr, r winRect, color uint32, px func(int) int32) {
	x, y := (r.Left+r.Right)/2, (r.Top+r.Bottom)/2
	pen := gcall("CreatePen", 0, uintptr(px(1)), uintptr(color))
	oldPen := gcall("SelectObject", dc, pen)
	oldBrush := gcall("SelectObject", dc, gcall("GetStockObject", 5))
	gcall("Ellipse", dc, uintptr(x-px(9)), uintptr(y-px(5)), uintptr(x+px(9)), uintptr(y+px(5)))
	gcall("Ellipse", dc, uintptr(x-px(2)), uintptr(y-px(2)), uintptr(x+px(2)), uintptr(y+px(2)))
	gcall("MoveToEx", dc, uintptr(x-px(8)), uintptr(y-px(8)), 0)
	gcall("LineTo", dc, uintptr(x+px(8)), uintptr(y+px(8)))
	gcall("SelectObject", dc, oldBrush)
	gcall("SelectObject", dc, oldPen)
	gcall("DeleteObject", pen)
}

func (n *nativeDesktop) paintCheckbox(h, dc uintptr) {
	c := n.controls[h]
	var r winRect
	call("GetClientRect", h, uintptr(unsafe.Pointer(&r)))
	call("FillRect", dc, uintptr(unsafe.Pointer(&r)), n.surfaceBrush)
	dpi := int(call("GetDpiForWindow", h))
	if dpi == 0 {
		dpi = 96
	}
	px := func(v int) int32 { return int32(v * dpi / 96) }
	t := n.theme
	fg, border, fill := t.text, t.border, t.surface
	checked := call("SendMessageW", h, 0xf0, 0, 0) != 0
	if checked {
		fill, border = t.accent, t.accent
	}
	if n.isHovered(h) && !t.contrast {
		fill = blendColor(fill, t.text, fluentHoverMix)
	}
	if call("SendMessageW", h, 0xf2, 0, 0)&4 != 0 && !t.contrast {
		fill = blendColor(fill, t.text, fluentPressedMix)
	}
	if call("IsWindowEnabled", h) == 0 {
		fg, border, fill = t.muted, t.muted, t.surface
	}
	box := winRect{px(2), (r.Bottom - px(18)) / 2, px(20), (r.Bottom + px(18)) / 2}
	n.roundRect(dc, box, fill, border, int(px(4)))
	if checked {
		checkColor := t.onAccent
		if call("IsWindowEnabled", h) == 0 {
			checkColor = t.muted
		}
		pen := gcall("CreatePen", 0, uintptr(px(2)), uintptr(checkColor))
		old := gcall("SelectObject", dc, pen)
		gcall("MoveToEx", dc, uintptr(box.Left+px(4)), uintptr(box.Top+px(9)), 0)
		gcall("LineTo", dc, uintptr(box.Left+px(7)), uintptr(box.Top+px(12)))
		gcall("LineTo", dc, uintptr(box.Left+px(14)), uintptr(box.Top+px(5)))
		gcall("SelectObject", dc, old)
		gcall("DeleteObject", pen)
	}
	r.Left = px(30)
	n.drawText(dc, n.fontFor(c.parent, 0), windowText(h), r, fg, 0x8024)
	if call("GetFocus") == h && n.keyboardFocusVisible() {
		r.Left = 0
		call("DrawFocusRect", dc, uintptr(unsafe.Pointer(&r)))
	}
}

func (n *nativeDesktop) paintInputFrame(h uintptr, combo bool, dc uintptr) {
	var r winRect
	call("GetWindowRect", h, uintptr(unsafe.Pointer(&r)))
	r.Right -= r.Left
	r.Bottom -= r.Top
	r.Left, r.Top = 0, 0
	dpi := int(call("GetDpiForWindow", h))
	if dpi == 0 {
		dpi = 96
	}
	px := func(v int) int32 { return int32(v * dpi / 96) }
	if dc == 0 {
		dc = call("GetWindowDC", h)
		defer call("ReleaseDC", h, dc)
	}
	border := n.theme.border
	if n.keyboardFocusVisible() && (call("GetFocus") == h || call("IsChild", h, call("GetFocus")) != 0) {
		border = n.theme.accent
	}
	if combo {
		arrow := winRect{r.Right - px(28), px(1), r.Right - px(1), r.Bottom - px(1)}
		call("FillRect", dc, uintptr(unsafe.Pointer(&arrow)), n.inputBrush)
		x, y := r.Right-px(14), r.Bottom/2-px(2)
		pen := gcall("CreatePen", 0, uintptr(px(2)), uintptr(n.theme.muted))
		old := gcall("SelectObject", dc, pen)
		gcall("MoveToEx", dc, uintptr(x-px(4)), uintptr(y), 0)
		gcall("LineTo", dc, uintptr(x), uintptr(y+px(4)))
		gcall("LineTo", dc, uintptr(x+px(4)), uintptr(y))
		gcall("SelectObject", dc, old)
		gcall("DeleteObject", pen)
	} else {
		// NCCALCSIZE reserves text padding; paint that non-client strip too
		// so the native edit does not leave a black seam above its contents.
		inset := winRect{1, 1, r.Right - 1, px(4)}
		call("FillRect", dc, uintptr(unsafe.Pointer(&inset)), n.inputBrush)
	}
	// Cover the native rectangular frame, then draw a restrained rounded outline.
	call("FrameRect", dc, uintptr(unsafe.Pointer(&r)), n.surfaceBrush)
	if combo {
		inner := winRect{1, 1, r.Right - 1, r.Bottom - 1}
		call("FrameRect", dc, uintptr(unsafe.Pointer(&inner)), n.surfaceBrush)
	}
	pen := gcall("CreatePen", 0, 1, uintptr(border))
	oldPen := gcall("SelectObject", dc, pen)
	oldBrush := gcall("SelectObject", dc, gcall("GetStockObject", 5)) // hollow brush
	gcall("RoundRect", dc, 0, 0, uintptr(r.Right), uintptr(r.Bottom), uintptr(px(fluentControlRadius*2)), uintptr(px(fluentControlRadius*2)))
	gcall("SelectObject", dc, oldPen)
	gcall("SelectObject", dc, oldBrush)
	gcall("DeleteObject", pen)
}
