package ui

import (
	"clipare/internal/config"
	"clipare/internal/history"
	"testing"
	"time"
)

func TestHistoryRowsGrouping(t *testing.T) {
	now := time.Date(2026, 10, 12, 18, 0, 0, 0, time.UTC)
	items := []history.Item{
		{ID: "a", OriginName: "MacBook", Timestamp: now.Unix(), Text: "today"},
		{ID: "b", OriginName: "Desktop", Timestamp: now.AddDate(0, 0, -1).Unix(), Text: "yesterday"},
		{ID: "c", OriginName: "Laptop", Timestamp: now.AddDate(0, 0, -5).Unix(), Text: "earlier"},
	}
	rows := historyRows(items, now, nil, nil)
	if len(rows) != 6 {
		t.Fatal("expected 3 items plus 3 group rows", rows)
	}
	groups := []string{}
	for _, row := range rows {
		if row.group {
			groups = append(groups, row.header)
		}
	}
	if groups[0] != "Сегодня" || groups[1] != "Вчера" || groups[2] != "Ранее" {
		t.Fatal("unexpected groups", groups)
	}
	if rows[1].header != "MacBook · "+time.Unix(now.Unix(), 0).Format("15:04") {
		t.Fatal("unexpected header", rows[1].header)
	}
}

func TestHistoryRowsMasking(t *testing.T) {
	now := time.Now()
	secret := history.Item{ID: "s", OriginName: "Desktop", Timestamp: now.Unix(), Type: history.ContentText, Sensitive: true, Text: "ghp_0123456789abcdefghijklmnopqrstuvwxyzAB"}
	rows := historyRows([]history.Item{secret}, now, nil, nil)
	if rows[1].preview != historyMaskedText {
		t.Fatal("secret not masked", rows[1].preview)
	}
	rows = historyRows([]history.Item{secret}, now, map[string]bool{"s": true}, nil)
	if rows[1].preview == historyMaskedText || rows[1].preview == "" {
		t.Fatal("revealed secret still masked", rows[1].preview)
	}
	plain := history.Item{ID: "p", OriginName: "Desktop", Timestamp: now.Unix(), Text: "обычный текст"}
	rows = historyRows([]history.Item{plain}, now, map[string]bool{"p": true}, nil)
	if rows[1].preview != "обычный текст" {
		t.Fatal("plain text changed", rows[1].preview)
	}
}

func TestHistoryRowsURL(t *testing.T) {
	now := time.Now()
	item := history.Item{ID: "u", OriginName: "Desktop", Timestamp: now.Unix(), Type: history.ContentURL, Text: "https://github.com/SawGoD/clipare"}
	rows := historyRows([]history.Item{item}, now, nil, nil)
	if !rows[1].url || rows[1].link != "https://github.com/SawGoD/clipare" {
		t.Fatal("url row misclassified", rows[1])
	}
}

func TestHistoryPreview(t *testing.T) {
	// Short text fits fully: no ellipsis, no expand control.
	short, truncated, height := historyPreview("Обрати внимание на записи", false)
	if short != "Обрати внимание на записи" || truncated {
		t.Fatal("short text truncated", short, truncated)
	}
	if height != historyRowHeight {
		t.Fatal("collapsed height", height)
	}
	// Two lines within the cutoff stay untruncated.
	twoLines := "первая строка\nвторая строка"
	if p, tr, _ := historyPreview(twoLines, false); p != twoLines || tr {
		t.Fatal("two-line text truncated", p, tr)
	}
	// Three lines exceed the two-line preview and become expandable.
	long := "первая строка\nвторая строка\nтретья строка"
	p, tr, _ := historyPreview(long, false)
	if p != long || !tr {
		t.Fatal("expected full preview with expand flag", p, tr)
	}
	// A single continuous token past the cutoff is expandable.
	runes, tr, _ := historyPreview("0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789", false)
	if len([]rune(runes)) != 130 || !tr {
		t.Fatal("continuous token capped or not expandable", len([]rune(runes)), tr)
	}
	if empty, _, _ := historyPreview("   \n\n  ", false); empty != "" {
		t.Fatal("empty preview", empty)
	}
	full, tr, expandedH := historyPreview(long, true)
	if full != long || !tr {
		t.Fatal("expanded preview mismatch", full)
	}
	if expandedH <= historyRowHeight {
		t.Fatal("expanded height not larger", expandedH)
	}
}

func TestRetentionMapping(t *testing.T) {
	if RetentionToLabel("24h") != "1 день" || RetentionToLabel("30m") != "30 минут" {
		t.Fatal("duration to label failed")
	}
	for _, duration := range config.HistoryRetentions {
		if RetentionFromLabel(RetentionToLabel(duration)) != duration {
			t.Fatalf("label roundtrip failed for %s", duration)
		}
	}
	if RetentionFromLabel("бессрочно") != "" || RetentionToLabel("5h") != "" {
		t.Fatal("unknown retention accepted")
	}
}

func TestSelectedHistoryItem(t *testing.T) {
	items := map[string]history.Item{"a": {ID: "a", Text: "x"}}
	rowIDs := []string{"", "a", ""}
	if item, ok := selectedHistoryItem(rowIDs, items, 1); !ok || item.ID != "a" {
		t.Fatal("item not resolved")
	}
	if _, ok := selectedHistoryItem(rowIDs, items, 0); ok {
		t.Fatal("group row resolved as item")
	}
	if _, ok := selectedHistoryItem(rowIDs, items, 5); ok {
		t.Fatal("out of range resolved")
	}
	if _, ok := selectedHistoryID(rowIDs, 0); ok {
		t.Fatal("group row has id")
	}
}
