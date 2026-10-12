package ui

import (
	"clipare/internal/config"
	"clipare/internal/history"
	"errors"
	"fmt"
	"strconv"
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
// and expanded flags are runtime-only UI state owned by the shared loop.
type historyRow struct {
	id        string
	header    string
	preview   string
	link      string // full URL for url rows
	height    int    // logical row height; adapters scale it
	sensitive bool
	revealed  bool
	expanded  bool
	truncated bool // expandable: content did not fit the collapsed preview
	url       bool
	group     bool
}

// viewDesktop lets the shared loop react to the native navigation state.
type viewDesktop interface{ CurrentView() ViewState }

const (
	historyGroupHeight  = 30
	historyRowHeight    = 64
	historyMaxExpand    = 560
	historyExpandCutoff = 128
	// Rough line capacity used to decide whether content exceeds the
	// two-line preview; adapters do the real word wrapping.
	historyLineCapacity = 80
)

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

// historyRows renders items (newest first) into display rows with date groups.
// Hidden sensitive rows are masked; reveal and expansion state is runtime-only.
func historyRows(items []history.Item, now time.Time, revealed, expanded map[string]bool) []historyRow {
	rows := make([]historyRow, 0, len(items)+3)
	lastGroup := ""
	for _, item := range items {
		label := historyGroupLabel(time.Unix(item.Timestamp, 0), now)
		if label != lastGroup {
			rows = append(rows, historyRow{group: true, header: label, height: historyGroupHeight})
			lastGroup = label
		}
		isRevealed := item.Sensitive && revealed[item.ID]
		// Secrets never offer a manual expand control: hidden rows stay
		// masked and collapsed; revealing with the eye expands them fully
		// right away.
		isExpanded := (!item.Sensitive && expanded[item.ID]) || (item.Sensitive && isRevealed)
		preview, truncated, height := historyPreview(item.Text, isExpanded)
		if item.Sensitive {
			truncated = false
			if !isRevealed {
				preview = historyMaskedText
			}
		}
		name := item.OriginName
		link := ""
		if item.Type == history.ContentURL {
			link = item.Text
		}
		if name == "" {
			name = item.OriginID
		}
		rows = append(rows, historyRow{
			id:        item.ID,
			header:    fmt.Sprintf("%s · %s", name, time.Unix(item.Timestamp, 0).Format("15:04")),
			preview:   preview,
			link:      link,
			height:    height,
			sensitive: item.Sensitive,
			revealed:  isRevealed,
			expanded:  isExpanded,
			truncated: truncated,
			url:       item.Type == history.ContentURL,
		})
	}
	return rows
}

// historyPreview renders one row's content. Collapsed rows show at most two
// word-wrapped lines (the cap itself is native); texts within the cutoff fit
// fully and never get an ellipsis, longer content becomes expandable. It
// reports whether the row offers an expand control and how tall the row is.
func historyPreview(text string, expanded bool) (preview string, truncated bool, height int) {
	trimmed := strings.TrimSpace(text)
	runes := []rune(trimmed)
	truncated = len(runes) > historyExpandCutoff || wrappedLines(trimmed) > 2
	if expanded {
		if len(runes) > 4000 {
			// Gigantic clips stay bounded even when expanded.
			return string(runes[:4000]) + "…", truncated, expandedHeight(string(runes[:4000]))
		}
		if trimmed == "" {
			return "…", truncated, historyRowHeight
		}
		return trimmed, truncated, expandedHeight(trimmed)
	}
	if len(runes) > 2000 {
		// The preview is capped natively to two lines; keep the payload sane.
		trimmed = string(runes[:2000])
	}
	return trimmed, truncated, historyRowHeight
}

// wrappedLines estimates how many display lines a text needs after word wrap.
func wrappedLines(text string) int {
	lines := 0
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if r := len([]rune(line)); r <= historyLineCapacity {
			lines++
		} else {
			lines += (r + historyLineCapacity - 1) / historyLineCapacity
		}
	}
	if lines == 0 {
		lines = 1
	}
	return lines
}

func expandedHeight(text string) int {
	lines := wrappedLines(text)
	if lines < 2 {
		lines = 2
	}
	height := 40 + lines*17
	if height > historyMaxExpand {
		height = historyMaxExpand
	}
	return height
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

// Only the two history controls participate; unrelated unsaved fields must
// never be validated or persisted by an immediate preference change.
func historySettingsFromForm(f form, current config.History) (config.History, error) {
	n, err := strconv.Atoi(strings.TrimSpace(f.Values[11]))
	if err != nil || n < config.HistoryMinItems || n > config.HistoryMaxItems {
		return current, errors.New("максимум записей должен быть от 3 до 50")
	}
	retention := RetentionFromLabel(f.Values[12])
	if retention == "" && f.Values[12] == "" {
		retention = current.Retention
	}
	if retention == "" {
		return current, errors.New("выберите срок хранения")
	}
	current.MaxItems, current.Retention = n, retention
	return current, nil
}
