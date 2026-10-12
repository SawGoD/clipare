package ui

import (
	"clipare/internal/config"
	"clipare/internal/history"
	"fmt"
	"strings"
	"time"
)

const (
	historyEmptyTitle   = "История пуста"
	historyEmptyHint    = "Скопированные и полученные элементы появятся здесь автоматически."
	historyKeyLostText  = "Не удалось расшифровать историю: ключ хранилища изменился. Очистите историю, чтобы начать заново."
	historyMaskedText   = "••••••••••••••••••••"
	historyClearTitle   = "Очистить историю?"
	historyClearConfirm = "Все сохранённые записи буфера обмена будут удалены только на этом устройстве."
)

// RetentionLabels lists user-facing retention presets, matching config.HistoryRetentions.
var RetentionLabels = []string{"30 минут", "1 час", "6 часов", "12 часов", "1 день", "2 дня", "3 дня"}

func RetentionToLabel(duration string) string {
	switch duration {
	case "30m":
		return RetentionLabels[0]
	case "1h":
		return RetentionLabels[1]
	case "6h":
		return RetentionLabels[2]
	case "12h":
		return RetentionLabels[3]
	case "24h":
		return RetentionLabels[4]
	case "48h":
		return RetentionLabels[5]
	case "72h":
		return RetentionLabels[6]
	}
	return ""
}

// RetentionFromLabel maps a user-facing label back to the canonical duration.
// Unknown values return "" so callers can keep the previous setting.
func RetentionFromLabel(label string) string {
	for i, known := range RetentionLabels {
		if known == label {
			return config.HistoryRetentions[i]
		}
	}
	if RetentionToLabel(label) != "" {
		return label // already canonical
	}
	return ""
}

// historyRow is one rendered line of the History list. Group rows carry
// section labels (Сегодня / Вчера / Ранее) instead of content. The revealed
// flag is runtime-only UI state owned by the shared loop.
type historyRow struct {
	id        string
	header    string
	preview   string
	link      string // full URL for url rows
	sensitive bool
	revealed  bool
	url       bool
	group     bool
}

func historyGroupLabel(moment, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case !moment.Before(today):
		return "Сегодня"
	case !moment.Before(today.AddDate(0, 0, -1)):
		return "Вчера"
	default:
		return "Ранее"
	}
}

// historyPreview caps content to three short lines with an ellipsis.
func historyPreview(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if r := []rune(line); len(r) > 80 {
			line = string(r[:80]) + "…"
		}
		out = append(out, line)
		if len(out) == 3 {
			break
		}
	}
	result := strings.Join(out, "\n")
	if result == "" {
		return "…"
	}
	if len(lines) > 3 {
		result += "\n…"
	}
	return result
}

// historyRows renders items (newest first) into display rows with date groups.
// Hidden sensitive rows are masked; reveal state is runtime-only.
func historyRows(items []history.Item, now time.Time, revealed map[string]bool) []historyRow {
	rows := make([]historyRow, 0, len(items)+3)
	lastGroup := ""
	for _, item := range items {
		label := historyGroupLabel(time.Unix(item.Timestamp, 0), now)
		if label != lastGroup {
			rows = append(rows, historyRow{group: true, header: label})
			lastGroup = label
		}
		isRevealed := item.Sensitive && revealed[item.ID]
		preview := historyPreview(item.Text)
		if item.Sensitive && !isRevealed {
			preview = historyMaskedText
		}
		name := item.OriginName
		if name == "" {
			name = item.OriginID
		}
		rows = append(rows, historyRow{
			id:        item.ID,
			header:    fmt.Sprintf("%s · %s", name, time.Unix(item.Timestamp, 0).Format("15:04")),
			preview:   preview,
			link:      item.Text,
			sensitive: item.Sensitive,
			revealed:  isRevealed,
			url:       item.Type == history.ContentURL,
		})
	}
	return rows
}

// historySummary builds the accessibility text of one row.
func historySummary(row historyRow) string {
	if row.group {
		return row.header
	}
	return row.header + ": " + strings.ReplaceAll(row.preview, "\n", " ")
}

// selectedHistoryItem resolves a list selection into a history item.
func selectedHistoryItem(rowIDs []string, items map[string]history.Item, index int) (history.Item, bool) {
	if index < 0 || index >= len(rowIDs) {
		return history.Item{}, false
	}
	item, ok := items[rowIDs[index]]
	return item, ok
}

// selectedHistoryID resolves a list selection into a row ID; group rows have
// no ID and never resolve.
func selectedHistoryID(rowIDs []string, index int) (string, bool) {
	if index < 0 || index >= len(rowIDs) || rowIDs[index] == "" {
		return "", false
	}
	return rowIDs[index], true
}

type historyDesktop interface {
	// ShowHistory renders the given rows and opens the History view.
	ShowHistory(rows []historyRow, keyLost bool)
	// HistoryEnabled reports the native "Сохранять историю" checkbox state.
	HistoryEnabled() bool
}

type historyPrefsDesktop interface {
	// HistoryPrefs restores native history settings controls from config.
	HistoryPrefs(c config.History)
}

func renderHistoryPrefs(d desktop, c config.History) {
	if n, ok := d.(historyPrefsDesktop); ok {
		n.HistoryPrefs(c)
	}
}

// historyFormValues extends the shared form with history settings fields.
func historyFormValues(c config.Config) (maxItems, retention string) {
	return fmt.Sprintf("%d", c.HistoryMaxItems()), RetentionToLabel(c.History.Retention)
}
