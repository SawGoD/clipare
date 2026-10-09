//go:build windows

package ui

func (n *nativeDesktop) installAdvanced() {
	n.window = n.home
	n.advancedBase = make(map[uintptr]int)
	previous := make(map[uintptr]bool)
	for h := range n.controls {
		previous[h] = true
	}
	n.label("Device ID", 40, 0, 420)
	n.helpButton(40, 492, 0, "Постоянный ID этого компьютера. Создан автоматически; менять его не нужно.")
	n.input(1, 40, 28, 488, false)
	call("SendMessageW", n.fields[1], 0xcf, 1, 0)
	n.label("NetBird IP / listen address", 40, 76, 300)
	n.helpButton(41, 344, 76, "Локальный NetBird IP, на котором Clipare принимает соединения. Не используйте публичный IP или 0.0.0.0.")
	n.fields[2] = n.control("COMBOBOX", "", 0x00210252, 40, 104, 340, 200, 102)
	n.styleCombo(n.fields[2])
	n.label("Порт", 396, 76, 132)
	n.input(3, 396, 104, 132, false)
	n.fields[5] = n.control("STATIC", "", 0x4000, 40, 152, 488, 44, 0)
	n.setRole(n.fields[5], 4)
	n.label("Legacy общий ключ", 40, 252, 488)
	n.helpButton(42, 492, 252, "Только для старого подключения по коду. Автоматический pairing использует отдельные ключи. Новый legacy key может нарушить старые соединения.")
	n.input(4, 40, 280, 488, true)
	n.button("Создать legacy key", 40, 324, 224, 5)
	n.button("Скопировать код", 280, 324, 248, 6)
	n.label("Имя другого компьютера", 40, 376, 236)
	n.input(6, 40, 404, 236, false)
	n.label("ID другого компьютера", 292, 376, 236)
	n.input(7, 292, 404, 236, false)
	n.label("IP / FQDN", 40, 448, 340)
	n.input(8, 40, 476, 340, false)
	n.label("Порт", 396, 448, 132)
	n.input(9, 396, 476, 132, false)
	n.button("Добавить / изменить peer", 40, 520, 488, 8)
	n.label("Код подключения", 40, 568, 440)
	n.helpButton(43, 492, 568, "Вставьте полный clipare1:… с другого компьютера. Код содержит legacy ключ: передавайте его приватно. Обычно удобнее подключить устройство через поиск.")
	n.input(10, 40, 596, 488, true)
	n.button("Добавить по коду", 40, 640, 240, 7)
	n.button("Применить", 288, 640, 240, 1)
	for h, c := range n.controls {
		if !previous[h] && c.parent == n.home {
			if c.bounds.y >= 252 || call("GetDlgCtrlID", h) == 42 {
				c.bounds.y -= 48
				n.controls[h] = c
			}
			n.advancedControls = append(n.advancedControls, h)
			n.advancedBase[h] = c.bounds.y
			visible(h, false)
		}
	}
}

func (n *nativeDesktop) helpButton(id, x, y int, text string) {
	if n.helpTexts == nil {
		n.helpTexts = make(map[int]string)
	}
	n.helpTexts[id] = text
	n.iconButton(text, "\ue897", x, y-6, 32, id)
}
