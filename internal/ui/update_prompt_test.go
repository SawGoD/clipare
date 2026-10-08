package ui

import (
	"strings"
	"testing"
)

func TestUpdatePromptStates(t *testing.T) {
	current := "0.4.1"
	cases := []struct {
		name             string
		prompt           updatePrompt
		primary, dismiss string
		action           int
	}{
		{"latest", updateNotice("Установлена последняя версия Clipare", current), "", "Понятно", 0},
		{"available", updateOffer(current, "0.5.0"), "Обновить", "Позже", eventInstallUpdate},
		{"check error", updateFailure("GitHub недоступен", current, eventCheckUpdate), "Повторить", "Закрыть", eventCheckUpdate},
		{"install error", updateFailure("Текущая версия продолжает работать", current, eventInstallUpdate), "Повторить", "Закрыть", eventInstallUpdate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := c.prompt
			if p.Primary != c.primary || p.Dismiss != c.dismiss || p.Action != c.action || p.Busy {
				t.Fatalf("unexpected prompt: %+v", p)
			}
			if !strings.Contains(p.Text, current) {
				t.Fatal("current version missing")
			}
		})
	}
	for _, message := range []string{"Проверка обновлений…", "Загрузка Clipare 0.5.0…", "Обновление готово"} {
		p := updateProgress(message)
		if !p.Busy || p.Primary != "" || p.Dismiss != "" || p.Action != 0 {
			t.Fatalf("busy prompt allows an action: %+v", p)
		}
	}
}
