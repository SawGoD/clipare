//go:build windows

package ui

func visible(h uintptr, show bool) {
	mode := uintptr(0)
	if show {
		mode = 5
	}
	call("ShowWindow", h, mode)
}

func (n *nativeDesktop) showDevicePresentation(count int) {
	s := presentSettings(count, n.preferencesExpanded)
	visible(n.emptyAdd, s.EmptyDevices)
	visible(n.emptyDevices, s.EmptyDevices)
	visible(n.homeList, s.ShowPeerList)
	visible(n.compactAdd, s.ShowCompactAdd)
	visible(n.removePeer, s.ShowRemove)
}

func (n *nativeDesktop) layoutPreferences() {
	s := presentSettings(len(n.rows[n.homeList]), n.preferencesExpanded)
	visible(n.homeAuto, s.ShowPreferences)
	visible(n.updateCheck, s.ShowPreferences)
	v := n.windows[n.home]
	v.cards[3].h = 48
	y := 620
	if s.ShowPreferences {
		v.cards[3].h = 136
		y = 708
	}
	c := n.controls[n.advancedButton]
	c.bounds.y = y
	n.controls[n.advancedButton] = c
	v.height = y + 68
	n.windows[n.home] = v
	n.fitWindow(n.home)
	n.layoutControls(n.home)
}
