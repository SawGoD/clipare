package ui

import (
	"clipare/internal/config"
	"strings"
)

type deviceRow struct {
	name, detail string
	online       bool
}

func pairedRows(peers []config.Peer, states map[string]bool) []deviceRow {
	rows := make([]deviceRow, 0, len(peers))
	for _, p := range peers {
		detail := "Не в сети"
		if states[p.ID] {
			detail = "Подключено"
		}
		rows = append(rows, deviceRow{peerLabel(p), detail, states[p.ID]})
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
		rows = append(rows, deviceRow{name, address, false})
	}
	return rows
}
