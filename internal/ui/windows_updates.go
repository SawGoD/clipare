//go:build windows

package ui

import (
	"clipare/internal/update"
	"unsafe"
)

const updateChannelID = 146

func (n *nativeDesktop) installUpdateChannel() {
	n.window = n.home
	before := make(map[uintptr]bool)
	for h := range n.controls {
		before[h] = true
	}
	n.label("Канал обновлений", 292, 552, 116)
	n.updateChannel = n.control("COMBOBOX", "", 0x00210252, 412, 548, 116, 160, updateChannelID)
	n.styleCombo(n.updateChannel)
	for _, label := range []string{"Стабильные", "Бета"} {
		call("SendMessageW", n.updateChannel, 0x143, 0, uintptr(unsafe.Pointer(wide(label))))
	}
	for h, c := range n.controls {
		if !before[h] && c.parent == n.home {
			n.historySettings = append(n.historySettings, h)
			n.historyBase[h] = c.bounds.y
			visible(h, false)
		}
	}
}

func (n *nativeDesktop) UpdateChannelSettings(channel string) {
	index := uintptr(0)
	if channel == update.ChannelBeta {
		index = 1
	}
	if call("SendMessageW", n.updateChannel, 0x147, 0, 0) != index {
		call("SendMessageW", n.updateChannel, 0x14e, index, 0)
	}
}
func (n *nativeDesktop) UpdateChannel() string {
	if call("SendMessageW", n.updateChannel, 0x147, 0, 0) == 1 {
		return update.ChannelBeta
	}
	return update.ChannelStable
}
