//go:build windows

package ui

import (
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var gdi = syscall.NewLazyDLL("gdi32.dll")
var dwm = syscall.NewLazyDLL("dwmapi.dll")
var themes = syscall.NewLazyDLL("uxtheme.dll")

func gcall(name string, args ...uintptr) uintptr {
	r, _, _ := gdi.NewProc(name).Call(args...)
	return r
}

func color(hex uint32) uint32 { return (hex&255)<<16 | hex&0xff00 | hex>>16 }

type winTheme struct {
	dark, contrast                                                             bool
	background, surface, text, muted, border, accent, onAccent, online, danger uint32
}

func registryDWORD(path, name string) (uint32, bool) {
	var value uint32
	size := uint32(4)
	r, _, _ := syscall.NewLazyDLL("advapi32.dll").NewProc("RegGetValueW").Call(
		0x80000001, uintptr(unsafe.Pointer(wide(path))), uintptr(unsafe.Pointer(wide(name))),
		0x10, 0, uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&size)))
	return value, r == 0
}

func systemTheme() winTheme {
	light, ok := registryDWORD(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "AppsUseLightTheme")
	t := winTheme{dark: ok && light == 0, background: color(0xf3f3f3), surface: color(0xffffff),
		text: color(0x1b1b1b), muted: color(0x616161), border: color(0xd8d8d8), accent: color(0x005fb8),
		onAccent: color(0xffffff), online: color(0x32754c), danger: color(0xa4262c)}
	if t.dark {
		t.background, t.surface = color(0x202020), color(0x2b2b2b)
		t.text, t.muted, t.border = color(0xf5f5f5), color(0xbcbcbc), color(0x494949)
		t.online, t.danger = color(0x8abf9b), color(0xffa7ab)
	}
	var accent uint32
	var opaque int32
	if r, _, _ := dwm.NewProc("DwmGetColorizationColor").Call(uintptr(unsafe.Pointer(&accent)), uintptr(unsafe.Pointer(&opaque))); r == 0 {
		t.accent = color(accent & 0xffffff)
	}
	// Choose readable primary-button text even for yellow or pale system accents.
	if contrastRatio(t.accent, color(0x000000)) > contrastRatio(t.accent, color(0xffffff)) {
		t.onAccent = 0
	}
	var hc struct {
		Size, Flags uint32
		Scheme      uintptr
	}
	hc.Size = uint32(unsafe.Sizeof(hc))
	if call("SystemParametersInfoW", 0x42, uintptr(hc.Size), uintptr(unsafe.Pointer(&hc)), 0) != 0 && hc.Flags&1 != 0 {
		t.contrast = true
		t.background = uint32(call("GetSysColor", 5))
		t.surface, t.text, t.muted = t.background, uint32(call("GetSysColor", 8)), uint32(call("GetSysColor", 8))
		t.accent, t.onAccent = uint32(call("GetSysColor", 13)), uint32(call("GetSysColor", 14))
		t.border, t.online, t.danger = t.text, t.text, t.text
	}
	return t
}

func (n *nativeDesktop) applyTheme() {
	n.setTheme(systemTheme())
}

func (n *nativeDesktop) setTheme(t winTheme) {
	n.theme = t
	for _, b := range []uintptr{n.surfaceBrush, n.backgroundBrush} {
		if b != 0 {
			gcall("DeleteObject", b)
		}
	}
	n.surfaceBrush = gcall("CreateSolidBrush", uintptr(n.theme.surface))
	n.backgroundBrush = gcall("CreateSolidBrush", uintptr(n.theme.background))
	for hwnd := range n.windows {
		dark := int32(0)
		if n.theme.dark && !n.theme.contrast {
			dark = 1
		}
		dwm.NewProc("DwmSetWindowAttribute").Call(hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)
		corners := int32(2)
		dwm.NewProc("DwmSetWindowAttribute").Call(hwnd, 33, uintptr(unsafe.Pointer(&corners)), 4)
		call("RedrawWindow", hwnd, 0, 0, 0x185)
	}
	for hwnd, c := range n.controls {
		if c.class == "EDIT" || c.class == "COMBOBOX" {
			name := "Explorer"
			if n.theme.dark {
				name = "DarkMode_Explorer"
			}
			themes.NewProc("SetWindowTheme").Call(hwnd, uintptr(unsafe.Pointer(wide(name))), 0)
		}
		call("RedrawWindow", hwnd, 0, 0, 0x185)
	}
}

func (n *nativeDesktop) fontFor(hwnd uintptr, role int) uintptr {
	dpi := int(call("GetDpiForWindow", hwnd))
	if dpi == 0 {
		dpi = 96
	}
	key := [2]int{dpi, role}
	if font := n.fonts[key]; font != 0 {
		return font
	}
	size, weight := 14, 400
	faceName := "Segoe UI"
	switch role {
	case 1:
		size, faceName = 20, "Segoe UI Semibold"
	case 2:
		size = 16
	case 3:
		size, faceName = 36, "Segoe UI Semibold"
	case 4:
		size = 12
	}
	create := func(face string) uintptr {
		var lf struct {
			Height, Width, Escapement, Orientation, Weight int32
			Flags                                          [8]byte
			Face                                           [32]uint16
		}
		lf.Height = -int32(size * dpi / 96)
		lf.Weight = int32(weight)
		lf.Flags[3] = 1
		lf.Flags[6] = 6 // CLEARTYPE_NATURAL_QUALITY, without synthesized bold.
		copy(lf.Face[:], syscall.StringToUTF16(face))
		font, _, _ := gdi.NewProc("CreateFontIndirectW").Call(uintptr(unsafe.Pointer(&lf)))
		runtime.KeepAlive(lf)
		return font
	}
	font := create(faceName)
	// GDI can silently substitute another family when the requested face is absent.
	// Check the realized face, not just the nonzero logical HFONT handle.
	if font != 0 {
		dc := gcall("CreateCompatibleDC", 0)
		old := gcall("SelectObject", dc, font)
		var face [128]uint16
		gcall("GetTextFaceW", dc, uintptr(len(face)), uintptr(unsafe.Pointer(&face[0])))
		gcall("SelectObject", dc, old)
		gcall("DeleteDC", dc)
		if !strings.HasPrefix(syscall.UTF16ToString(face[:]), "Segoe UI") {
			gcall("DeleteObject", font)
			font = create("Segoe UI")
		}
	}
	if font == 0 {
		font = gcall("GetStockObject", 17)
		return font
	}
	n.fonts[key] = font
	return font
}

func (n *nativeDesktop) themeMessage(hwnd uintptr, msg uint32, w, l uintptr) (uintptr, bool) {
	switch msg {
	case 0x2c: // WM_MEASUREITEM can arrive before the combo HWND is registered.
		var item struct {
			Type, ID, Item, Width, Height uint32
			Data                          uintptr
		}
		k32.NewProc("RtlMoveMemory").Call(uintptr(unsafe.Pointer(&item)), l, unsafe.Sizeof(item))
		if item.Type == 3 {
			item.Height = uint32(n.scaledRect(hwnd, logicalRect{h: 30}).Bottom)
			k32.NewProc("RtlMoveMemory").Call(l, uintptr(unsafe.Pointer(&item)), unsafe.Sizeof(item))
			return 1, true
		}
	case 0x1a, 0x31a, 0x320: // settings, theme, DWM accent changed
		n.applyTheme()
		return 0, true
	case 0x14: // background and cards, without classic grey label stripes
		var r winRect
		call("GetClientRect", hwnd, uintptr(unsafe.Pointer(&r)))
		call("FillRect", w, uintptr(unsafe.Pointer(&r)), n.backgroundBrush)
		for _, card := range n.windows[hwnd].cards {
			r := n.scaledRect(hwnd, card)
			v := n.windows[hwnd]
			r.Left -= v.scrollX
			r.Right -= v.scrollX
			r.Top -= v.scrollY
			r.Bottom -= v.scrollY
			n.roundRect(w, r, n.theme.surface, n.theme.border, int(n.scaledRect(hwnd, logicalRect{w: 8}).Right))
		}
		return 1, true
	case 0x133, 0x134, 0x135, 0x138: // edit/list/button/static colors
		gcall("SetTextColor", w, uintptr(n.theme.text))
		brush, bg := n.backgroundBrush, n.theme.background
		c := n.controls[l]
		if c.role == 4 {
			gcall("SetTextColor", w, uintptr(n.theme.muted))
		}
		for _, r := range n.windows[c.parent].cards {
			if c.bounds.x >= r.x && c.bounds.y >= r.y && c.bounds.x < r.x+r.w && c.bounds.y < r.y+r.h {
				brush, bg = n.surfaceBrush, n.theme.surface
				break
			}
		}
		if c.class == "EDIT" || c.class == "COMBOBOX" || c.class == "LISTBOX" {
			brush, bg = n.surfaceBrush, n.theme.surface
		}
		gcall("SetBkColor", w, uintptr(bg))
		return brush, true
	case 0x2b:
		var item drawItem
		k32.NewProc("RtlMoveMemory").Call(uintptr(unsafe.Pointer(&item)), l, unsafe.Sizeof(item))
		n.drawItem(item)
		return 1, true
	}
	return 0, false
}
