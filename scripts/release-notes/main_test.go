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
