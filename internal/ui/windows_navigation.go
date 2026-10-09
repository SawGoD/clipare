//go:build windows

package ui

import (
	"clipare/internal/instance"
	"unsafe"
)

func (n *nativeDesktop) Instance(path string) {
	n.instanceKey = instance.Key(path)
	n.activation = call("RegisterWindowMessageW", uintptr(unsafe.Pointer(wide(n.instanceKey))))
}
func (n *nativeDesktop) requestOpen() {
	if n.prepared {
		n.navigate(n.navigation.View)
	} else {
		n.events = append(n.events, eventSettings)
	}
}

func (n *nativeDesktop) activePanel() uintptr {
	switch n.navigation.Container() {
	case ViewUpdate:
		return n.updateWindow
	case ViewNotice:
		return n.notice
	default:
		return n.home
	}
}

func (n *nativeDesktop) sizePanel(h uintptr) {
	var r winRect
	call("GetClientRect", n.main, uintptr(unsafe.Pointer(&r)))
	call("SetWindowPos", h, 0, 0, 0, uintptr(r.Right), uintptr(r.Bottom), 0x14)
}

func (n *nativeDesktop) navigate(v ViewState) {
	previous := n.navigation.View
	n.navigation.Open(v)
	if call("IsIconic", n.main) != 0 {
		call("ShowWindow", n.main, 9) // SW_RESTORE, not SW_SHOW on a minimized HWND.
	}
	for _, h := range []uintptr{n.home, n.updateWindow, n.notice} {
		visible(h, h == n.activePanel())
	}
	if n.activePanel() == n.home {
		n.showDevicePresentation(len(n.rows[n.homeList]))
		n.layoutPreferences()
	} else {
		n.fitWindow(n.activePanel())
		n.layoutControls(n.activePanel())
	}
	visible(n.main, true)
	call("SetForegroundWindow", n.main)
	// Focus never remains in a hidden panel.
	focus := call("GetFocus")
	if previous != v || focus == 0 || call("IsWindowVisible", focus) == 0 {
		call("SetFocus", n.activePanel())
	}
}

func (n *nativeDesktop) back(closeWindow bool) {
	switch n.navigation.View {
	case ViewNotice:
		n.navigate(n.navigation.DismissNotice())
		if closeWindow {
			visible(n.main, false)
		}
		return
	case ViewPairing:
		if n.pairIncoming {
			n.events = append(n.events, eventReject)
		} else {
			n.events = append(n.events, eventCancelPair)
		}
	case ViewDiscovery:
		n.events = append(n.events, eventCloseDiscovery)
	case ViewUpdate:
		if call("IsWindowVisible", n.updateDismiss) == 0 {
			return
		}
		n.events = append(n.events, eventLaterUpdate)
	}
	n.navigate(ViewHome)
	if closeWindow {
		visible(n.main, false)
	}
}
