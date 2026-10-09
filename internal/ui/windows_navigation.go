//go:build windows

package ui

import "unsafe"

func (n *nativeDesktop) activePanel() uintptr {
	switch n.navigation.View {
	case ViewDiscovery:
		return n.found
	case ViewPairing:
		return n.pair
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
	n.navigation.Open(v)
	for _, h := range []uintptr{n.home, n.found, n.pair, n.updateWindow, n.notice} {
		visible(h, h == n.activePanel())
	}
	n.fitWindow(n.activePanel())
	n.layoutControls(n.activePanel())
	visible(n.main, true)
	call("SetForegroundWindow", n.main)
	// Focus never remains in a hidden panel.
	call("SetFocus", n.activePanel())
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
