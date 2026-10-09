package ui

import (
	"clipare/internal/config"
	"testing"
)

func TestDeviceRows(t *testing.T) {
	rows := pairedRows([]config.Peer{{ID: "a", Name: "MacBook"}, {ID: "b", Name: "Laptop"}}, map[string]bool{"a": true})
	if len(rows) != 2 || !rows[0].online || rows[0].detail != "Подключено" || rows[1].online || rows[1].detail != "Не в сети" {
		t.Fatalf("wrong statuses: %+v", rows)
	}
	if rows[0].name != "MacBook" {
		t.Fatal("name changed")
	}
	if len(pairedRows(nil, nil)) != 0 {
		t.Fatal("phantom devices")
	}
	found := discoveredRows("Desktop — 100.64.0.2\nLaptop — laptop.netbird.cloud\n")
	if len(found) != 2 || found[0].name != "Desktop" || found[0].detail != "100.64.0.2" {
		t.Fatalf("wrong discovery rows: %+v", found)
	}
}

func TestPlatformsUseMetadataNotHostname(t *testing.T) {
	peers := []config.Peer{{ID: "win", Name: "MacBook"}, {ID: "mac", Name: "Windows-PC"}, {ID: "unknown", Name: "MacBook Pro"}}
	rows := pairedRows(peers, nil, map[string]string{"win": "windows", "mac": "darwin", "unknown": "unrecognized"})
	if rows[0].platform != "windows" || rows[1].platform != "darwin" || rows[2].platform != "" {
		t.Fatal("platform guessed or invalid metadata accepted")
	}
	for _, row := range pairedRows(peers, nil) {
		if row.platform != "" {
			t.Fatal("hostname inferred platform")
		}
	}
	if platformName("") != "Устройство" {
		t.Fatal("unknown lacks accessible fallback")
	}
}

func TestAccentContrast(t *testing.T) {
	if contrastRatio(0xffffff, 0) < 20 {
		t.Fatal("black and white contrast")
	}
	for _, accent := range []uint32{0, 0xffffff, 0x00ffff, 0xcc3300, 0xff00ff} {
		a, b := contrastRatio(accent, 0), contrastRatio(accent, 0xffffff)
		if a < 4.5 && b < 4.5 {
			t.Fatal("no readable foreground")
		}
	}
}
