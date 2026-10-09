package main

import (
	"strings"
	"testing"
)

func TestNotes(t *testing.T) {
	c := "# Changelog\n\n## 0.4.1\n\n### Исправления\n\n- Кнопка «Понятно».\n\n## 0.4.0\n\nold"
	n, err := notes(c, "0.4.1", "0.4.0")
	if err != nil || !strings.Contains(n, "## Исправления") || !strings.Contains(n, "v0.4.0...v0.4.1") || strings.Contains(n, "old") {
		t.Fatalf("%q %v", n, err)
	}
	for _, c := range []string{"", "## 0.4.1\n", "## 0.4.1\n- raw commit"} {
		if _, err := notes(c, "0.4.1", "0.4.0"); err == nil {
			t.Fatal("accepted missing curated summary")
		}
	}
	if _, err := notes(c, "0.4.1", "bad/version"); err == nil {
		t.Fatal("invalid comparison version")
	}
}

func TestBetaNotes(t *testing.T) {
	c := "## 0.5.0-beta.1\n\n### Нововведения\n\n- Preview интерфейса.\n"
	n, err := notes(c, "0.5.0-beta.1", "0.4.1")
	if err != nil || !strings.Contains(n, "v0.4.1...v0.5.0-beta.1") {
		t.Fatalf("%q %v", n, err)
	}
	if _, err = notes(c, "0.5.0-beta.01", "0.4.1"); err == nil {
		t.Fatal("accepted invalid prerelease")
	}
}
