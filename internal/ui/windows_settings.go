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
	if len(v.cards) > 4 {
		v.cards = v.cards[:4]
	}
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
	for _, h := range n.advancedControls {
		visible(h, n.navigation.AdvancedExpanded)
		c := n.controls[h]
		c.bounds.y = n.advancedBase[h] + y + 64
		n.controls[h] = c
	}
	if n.navigation.AdvancedExpanded {
		v.cards = append(v.cards, logicalRect{24, y + 48, 520, 808})
		v.height = y + 880
	}
	n.windows[n.home] = v
	n.fitWindow(n.home)
	n.layoutControls(n.home)
}
