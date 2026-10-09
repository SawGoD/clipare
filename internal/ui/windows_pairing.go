//go:build windows

package ui

func (n *nativeDesktop) installPairing() {
	n.pair, n.window = n.home, n.home
	before := make(map[uintptr]bool)
	for h := range n.controls {
		before[h] = true
	}
	n.iconButton("Отменить подключение", "\ue72b", 40, 224, 36, 31)
	n.heading("Подключение устройства", 88, 224, 440, 2)
	n.pairName = n.control("STATIC", "", 0x4000, 40, 264, 488, 28, 0)
	n.pairCode = n.control("STATIC", "", 1, 40, 304, 488, 48, 0)
	n.setRole(n.pairCode, 3)
	n.pairHelp = n.control("STATIC", "", 0, 40, 364, 488, 64, 0)
	n.setRole(n.pairHelp, 4)
	n.pairReject = n.control("BUTTON", "Отменить", 0x10000, 40, 440, 164, 32, 15)
	n.pairAllow = n.control("BUTTON", "Разрешить", 0x10000, 348, 440, 180, 32, 14)
	for h, c := range n.controls {
		if !before[h] && c.parent == n.home {
			n.pairControls = append(n.pairControls, h)
			visible(h, false)
		}
	}
}

func (n *nativeDesktop) layoutPairing() int {
	show := n.navigation.View == ViewPairing
	for _, h := range n.pairControls {
		visible(h, show && (h != n.pairAllow || n.pairMode != 0))
	}
	if show {
		return 56
	}
	return 0
}
