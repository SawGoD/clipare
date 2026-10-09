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
	home := n.navigation.View != ViewDiscovery
	visible(n.devicesHeader, home)
	visible(n.emptyAdd, home && s.EmptyDevices)
	visible(n.emptyDevices, home && s.EmptyDevices)
	visible(n.homeList, home && s.ShowPeerList)
	visible(n.compactAdd, home && s.ShowCompactAdd)
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
	delta := n.layoutDiscovery()
	v.cards[1].h = 228 + delta
	v.cards[2].y = 452 + delta
	for h, y := range map[uintptr]int{n.preferencesToggle: 460, n.homeAuto: 508, n.updateCheck: 548} {
		c := n.controls[h]
		c.bounds.y = y + delta
		n.controls[h] = c
	}
	y := 516 + delta
	if s.ShowPreferences {
		v.cards[2].h = 136
		y = 604 + delta
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
