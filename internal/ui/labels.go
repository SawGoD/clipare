package ui

import (
	"clipare/internal/config"
	"strconv"
	"strings"
)

func peerLabel(p config.Peer) string {
	name := p.Name
	if name == "" {
		name = p.ID
	}
	return strings.ReplaceAll(strings.ReplaceAll(name, "\n", " "), "\r", " ")
}
func peerValues(p config.Peer) [4]string {
	return [4]string{p.Name, p.ID, p.Address, strconv.Itoa(p.Port)}
}
func peerStatuses(peers []config.Peer, states map[string]bool) string {
	var lines []string
	for _, p := range peers {
		mark := "○ "
		if states[p.ID] {
			mark = "● "
		}
		lines = append(lines, mark+peerLabel(p))
	}
	if len(lines) == 0 {
		return "Нет устройств — откройте настройки"
	}
	return strings.Join(lines, "\n")
}
