//go:build windows

package ui

func visible(h uintptr, show bool) {
	if h == 0 || (call("GetWindowLongPtrW", h, ^uintptr(15))&0x10000000 != 0) == show {
		return
	}
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
}

func (n *nativeDesktop) layoutPreferences() {
	s := presentSettings(len(n.rows[n.homeList]), n.preferencesExpanded)
	visible(n.homeAuto, s.ShowPreferences)
	visible(n.updateCheck, s.ShowPreferences)
	v := n.windows[n.home]
	if len(v.cards) > 3 {
		v.cards = v.cards[:3]
	}
	v.cards[2].h = 48
	y := 516
	if s.ShowPreferences {
		v.cards[2].h = 136
		y = 604
	}
	c := n.controls[n.advancedButton]
	c.bounds.y = y
	n.controls[n.advancedButton] = c
	footer := y + 52
	for _, h := range n.advancedControls {
		visible(h, n.navigation.AdvancedExpanded)
		c := n.controls[h]
		c.bounds.y = n.advancedBase[h] + y + 64
		n.controls[h] = c
	}
	if n.navigation.AdvancedExpanded {
		v.cards = append(v.cards, logicalRect{24, y + 48, 520, 724})
		footer = y + 792
	}
	for _, h := range []uintptr{n.updateVersion, n.footerCheck} {
		c := n.controls[h]
		c.bounds.y = footer
		if h == n.updateVersion {
			c.bounds.y += 8
		}
		n.controls[h] = c
	}
	v.height = footer + 56
	n.windows[n.home] = v
	n.fitWindow(n.home)
	n.layoutControls(n.home)
}
