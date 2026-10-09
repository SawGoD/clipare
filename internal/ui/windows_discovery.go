//go:build windows

package ui

// The discovery controls share the home panel and occupy only its device card.
func (n *nativeDesktop) installDiscovery() {
	n.found = n.home
	n.window = n.home
	before := make(map[uintptr]bool)
	for h := range n.controls {
		before[h] = true
	}
	n.iconButton("Вернуться к устройствам", "\ue72b", 40, 224, 36, 31)
	n.heading("Найденные устройства", 88, 224, 440, 2)
	n.foundList = n.deviceList(40, 264, 488, 120, 132)
	n.emptyDiscovery = n.control("STATIC", discoveryEmptyMessage, 0, 40, 264, 488, 88, 0)
	n.foundStatus = n.control("STATIC", "Поиск устройств…", 0, 40, 392, 488, 40, 0)
	n.setRole(n.foundStatus, 4)
	n.iconButton("Обновить список устройств", "\ue72c", 40, 444, 36, eventRefresh)
	n.button("По коду…", 88, 444, 160, 30)
	n.button("Подключить", 384, 444, 144, eventConnect)
	for h, c := range n.controls {
		if !before[h] && c.parent == n.home {
			n.discoveryControls = append(n.discoveryControls, h)
			visible(h, false)
		}
	}
}

func (n *nativeDesktop) layoutDiscovery() int {
	show := n.navigation.View == ViewDiscovery
	for _, h := range n.discoveryControls {
		visible(h, show)
	}
	if !show {
		return 0
	}
	count := len(n.rows[n.foundList])
	height := count * 60
	if height > 180 {
		height = 180
	}
	if count == 0 {
		height = 88
	}
	visible(n.foundList, count > 0)
	visible(n.emptyDiscovery, count == 0)
	visible(n.foundStatus, count > 0)
	c := n.controls[n.foundList]
	c.bounds.h = height
	n.controls[n.foundList] = c
	for _, h := range n.discoveryControls {
		c := n.controls[h]
		if h == n.foundStatus {
			c.bounds.y = 264 + height + 8
		}
		if c.class == "BUTTON" && c.bounds.y != 224 {
			c.bounds.y = 264 + height + 60
		}
		n.controls[h] = c
	}
	return 164 + height - 228
}
