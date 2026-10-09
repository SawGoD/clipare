//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

// Real buttons keep row actions keyboard-accessible, with native names/tooltips.
// Reuse a pool instead of recreating controls on every health refresh.
func (n *nativeDesktop) updateRowTrash() {
	rows := n.rows[n.homeList]
	if n.rowCallback == 0 {
		n.rowCallback = syscall.NewCallback(func(h uintptr, msg uint32, w, l, id, ref uintptr) uintptr {
			result, _, _ := comctl.NewProc("DefSubclassProc").Call(h, uintptr(msg), w, l)
			if msg == 0x115 || msg == 0x20a || msg == 5 {
				n.positionRowTrash()
			}
			return result
		})
		comctl.NewProc("SetWindowSubclass").Call(n.homeList, n.rowCallback, 2, 0)
	}
	previous := n.window
	n.window = n.home
	for len(n.rowTrash) < len(rows) {
		index := len(n.rowTrash)
		h := n.iconButton("Удалить устройство", "\ue74d", 0, 0, 32, 9000+index)
		c := n.controls[h]
		c.kind = 2
		n.controls[h] = c
		n.rowTrash = append(n.rowTrash, h)
	}
	n.window = previous
	n.positionRowTrash()
}

func (n *nativeDesktop) positionRowTrash() {
	var client winRect
	call("GetClientRect", n.homeList, uintptr(unsafe.Pointer(&client)))
	for i, h := range n.rowTrash {
		if n.navigation.View == ViewDiscovery || n.navigation.View == ViewPairing {
			visible(h, false)
			continue
		}
		var r winRect
		if i >= len(n.rows[n.homeList]) || int32(call("SendMessageW", n.homeList, 0x198, uintptr(i), uintptr(unsafe.Pointer(&r)))) < 0 || r.Top < 0 || r.Bottom > client.Bottom {
			visible(h, false)
			continue
		}
		dpi := int(call("GetDpiForWindow", n.home))
		if dpi == 0 {
			dpi = 96
		}
		size := int32(32 * dpi / 96)
		x, y := client.Right-size-8, r.Top+(r.Bottom-r.Top-size)/2
		point := struct{ X, Y int32 }{x, y}
		call("MapWindowPoints", n.homeList, n.home, uintptr(unsafe.Pointer(&point)), 1)
		// Row buttons overlap the list's HWND: keep them above it for painting
		// and hit testing, including after selection/health refresh.
		call("SetWindowPos", h, 0, uintptr(point.X), uintptr(point.Y), uintptr(size), uintptr(size), 0x10)
		call("SendMessageW", h, 0x30, n.fontFor(n.home, 0), 0)
		visible(h, true)
	}
}

func (n *nativeDesktop) confirmRowRemoval(index int) {
	if index < 0 || index >= len(n.rows[n.homeList]) {
		return
	}
	name := n.rows[n.homeList][index].name
	result := call("MessageBoxW", n.main, uintptr(unsafe.Pointer(wide("Прекратить синхронизацию с устройством «"+name+"»?"))), uintptr(unsafe.Pointer(wide("Удалить устройство?"))), 0x124)
	if result == 6 {
		call("SendMessageW", n.homeList, 0x186, uintptr(index), 0)
		n.events = append(n.events, eventRemove)
	}
}
