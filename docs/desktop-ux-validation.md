# Приёмка Desktop UX

Это checklist для preview, а не отметка о выполненной ручной приёмке.

## Раздельные релизы

Последний опубликованный stable — v0.4.1; VERSION содержит ещё не опубликованную Windows-preview 0.5.0. Не публиковать все текущие изменения одним тегом.

1. Завершить приёмку Windows Fluent 0.5.0.
2. Release A: следующий MINOR с общим UX — один root window, inline Advanced, typed status и badges, системные action icons. Код этого этапа расположен до коммита `4f856f8` (который начинает Glass). При подготовке release branch включить последующие общие исправления, не включая macOS materials.
3. После Release A — Release B: следующий MINOR с macOS Glass и accessibility/appearance исправлениями.

Для каждого релиза отдельно обновить VERSION и CHANGELOG, использовать разделы «Нововведения»/«Исправления» и comparison с предыдущим stable. Публиковать только после CI и ручной приёмки. Preview ZIP в Actions не является stable release.

## Общие сценарии

- [ ] Ноль peers: большая кнопка добавления, нет пустой таблицы/удаления.
- [ ] Online/offline peers: список, компактный add; offline не меняет зелёный глобальный статус.
- [ ] Home → discovery → pairing → home: всё в одном окне, одинаковый SAS, reject/approve/cancel/timeout.
- [ ] Входящий pairing: одно уведомление, foreground существующего окна, нет нового окна.
- [ ] Inline notice возвращает к текущему workflow, а не теряет pairing.
- [ ] Updates: latest («Понятно» + версия), available («Позже»), download, restart-ready, error/retry.
- [ ] Advanced раскрывается/сворачивается в том же окне, показывает только технические поля.
- [ ] Autostart/updates применяются сразу; при отказе сохранения флажки возвращаются к сохранённым значениям.
- [ ] Active/paused/degraded: совпадают dot, tooltip, icon badge и menu text.
- [ ] Закрытие окна скрывает приложение; Quit завершает его корректно.

## Windows 11

- [ ] Light/Dark, разные system accents, Contrast themes.
- [ ] 100/125/150/200% DPI и перенос между мониторами.
- [ ] Короткий экран после длинного Advanced не оставляет scrollbars.
- [ ] Tab/Shift+Tab/Enter/Escape, focus ring, icon tooltip/accessibility name.
- [ ] Tray badge читается в обоих вариантах taskbar.

Workflow `Windows UI preview` проверяет native state и публикует ZIP/screenshots; тестовые палитры не заменяют проверку реальной системной темы.

## macOS

- [ ] Tahoe 26: реальный Glass, Light/Dark, несколько system accents.
- [ ] Reduce Transparency ON/OFF во время работы; непрозрачный режим сохраняет контраст.
- [ ] macOS 15/старый SDK: native material fallback, без поднятия deployment target.
- [ ] Retina/внешний монитор, длинные имена, компактный home и прокрутка Advanced.
- [ ] Tab/Shift+Tab/Return/Escape/Cmd+W/Cmd+Q и Cmd+Z/X/C/V/A.
- [ ] Menu bar mark адаптируется к appearance, badge остаётся цветным.
- [ ] Уведомление об incoming pairing при разрешении и отказе notification permission.

`make test-macos-ui` не взаимодействует с настоящим clipboard/config. Bitmap capture не полностью отображает системный GPU Glass; снимки нельзя считать подтверждением реального вида эффекта.

## Регрессии перед каждым stable

- [ ] Clipboard Mac ↔ Windows, Unicode, быстрые копирования, недоступный peer.
- [ ] Discovery, SAS, membership третьего устройства, удаление peer, legacy import.
- [ ] Settings persistence, autostart после входа, self-update с сохранением config.
- [ ] Shutdown во время запросов/поиска, отсутствие дублирующих процессов.
- [ ] `go test -race ./...`, `go vet ./...`, CI Linux/macOS/Windows.
- [ ] Сборки darwin-arm64/amd64 и windows-amd64/arm64.
