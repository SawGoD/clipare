package ui

// updatePrompt keeps action semantics independent of native widgets.
type updatePrompt struct {
	Text, Primary, Dismiss string
	Action                 int
	Busy                   bool
}

func updateNotice(message, version string) updatePrompt {
	return updatePrompt{Text: message + "\n\nТекущая версия: " + version, Dismiss: "Понятно"}
}

func updateFailure(message, version string, action int) updatePrompt {
	return updatePrompt{Text: "Не удалось обновить Clipare\n\n" + message + "\n\nТекущая версия: " + version, Primary: "Повторить", Dismiss: "Закрыть", Action: action}
}

func updateProgress(message string) updatePrompt {
	return updatePrompt{Text: message, Busy: true}
}

func updateOffer(current, next string) updatePrompt {
	return updatePrompt{Text: "Доступно обновление Clipare\n\nУстановлена: " + current + "\nДоступна: " + next + "\n\nПосле загрузки и проверки Clipare перезапустится. Настройки и устройства сохранятся.", Primary: "Обновить", Dismiss: "Позже", Action: eventInstallUpdate}
}
