package ui

import (
	"clipare/internal/config"
	"strings"
)

type deviceRow struct {
	name, detail string
	online       bool
	platform     string
}

type platformDesktop interface{ Platforms(map[string]string) }

func renderPlatforms(d desktop, session interface{ Platforms() map[string]string }) {
	if n, ok := d.(platformDesktop); ok {
		n.Platforms(session.Platforms())
	}
}
func platformName(value string) string {
	switch value {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	default:
		return "Устройство"
	}
}

func pairedRows(peers []config.Peer, states map[string]bool, platforms ...map[string]string) []deviceRow {
	rows := make([]deviceRow, 0, len(peers))
	for _, p := range peers {
		detail := "Не в сети"
		if states[p.ID] {
			detail = "Подключено"
		}
		os := ""
		if len(platforms) > 0 {
			os = platforms[0][p.ID]
		}
		if os != "windows" && os != "darwin" {
			os = ""
		}
		rows = append(rows, deviceRow{name: peerLabel(p), detail: detail, online: states[p.ID], platform: os})
	}
	return rows
}

func discoveredRows(lines string) []deviceRow {
	var rows []deviceRow
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		name, address, _ := strings.Cut(line, " — ")
		rows = append(rows, deviceRow{name: name, detail: address})
	}
	return rows
}
