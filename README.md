# Clipare

Минималистичная peer-to-peer синхронизация текстового буфера обмена через уже настроенную сеть NetBird. Go, без центрального сервера, аккаунтов, истории clipboard и браузерного UI.

Сейчас реализована **Phase 1: headless prototype** для Windows 11 и macOS. Tray/menu bar, окно настроек и автозапуск относятся к Phase 2 и будут добавлены после подтверждения работы на реальной паре Mac ↔ Windows. Мобильные устройства, картинки, файлы и HTML не поддерживаются.

## Сборка

Нужен Go 1.23+. Единственная внешняя runtime-зависимость Go — `go.yaml.in/yaml/v3 v3.0.5` для конфигурации: [форк официальной YAML-организации](https://github.com/yaml/go-yaml). Линия v3 получает security fixes, сохраняя API; CGO и системных зависимостей у парсера нет. Исходный `gopkg.in/yaml.v3` архивирован и заменён.

macOS (arm64 или amd64): установите Xcode Command Line Tools (`xcode-select --install`). Адаптер NSPasteboard написан на Objective-C, поэтому CGO включён; используется системный AppKit. Дополнительный runtime не требуется.

```sh
go build -o clipare ./cmd/clipare
```

Windows 11 (PowerShell, CGO не нужен):

```powershell
$env:CGO_ENABLED = "0"
go build -o clipare.exe ./cmd/clipare
```

Кросс-компиляция Windows с Mac:

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o clipare.exe ./cmd/clipare
```

Для Windows arm64 замените `GOARCH` на `arm64`. Сборку macOS выполняйте на macOS с соответствующим SDK. Linux не входит в поддерживаемые платформы; протокол и тесты могут запускаться без native clipboard.

## Настройка и запуск

Создайте `config.yaml` по [примеру](configs/config.example.yaml). Пример намеренно содержит невалидный `CHANGE_ME`: замените его на случайный общий secret длиной не менее 32 байт. Например, `openssl rand -hex 32` на Mac. Передайте его доверенным устройствам через безопасный канал. Ограничьте доступ к config текущим пользователем (на Mac `chmod 600 config.yaml`). Файл `config.yaml` исключён из Git; конфиги с секретами не коммитьте.

```yaml
device:
  id: macbook-home
  name: MacBook
listen:
  address: 100.100.100.101
  port: 45873
security:
  secret: CHANGE_ME
sync:
  enabled: true
  mode: bidirectional
peers:
  - id: desktop
    name: Desktop
    address: desktop.netbird.cloud
    port: 45873
```

На Windows задайте собственные `device.id`, `device.name`, NetBird IP в `listen.address`, а в `peers` — Mac. ID устройства задаётся явно и должен совпадать с его ID в конфигурациях остальных peers. Не используйте один ID на двух компьютерах. Secret должен совпадать на всех устройствах.

Для трёх и более компьютеров настройте полный mesh: каждый перечисляет все остальные. Полученные сообщения применяются локально, но не ретранслируются. Недоступный peer не получает накопленную историю после возвращения в сеть.

```sh
./clipare --config /absolute/path/config.yaml
./clipare --config /absolute/path/config.yaml --debug
./clipare status --config /absolute/path/config.yaml
./clipare --version
```

На Windows используйте `./clipare.exe`. Приложение работает в интерактивном пользовательском сеансе, где доступен clipboard. Начальное содержимое при запуске не отправляется: синхронизация начинается с нового копирования. Для остановки — Ctrl+C; на macOS также SIGTERM. Изменение конфигурации требует перезапуска.

`sync.enabled: false` либо `sync.mode: disabled` отключает отправку и приём; health остаётся доступным. Также реализованы `send-only` и `receive-only`; по умолчанию `bidirectional`. Отправитель делает одну попытку доставки, timeout 2 секунды. Каждый peer имеет свой worker и очередь на одно последнее ожидающее сообщение. Health проверяется раз в 20 секунд, статусы видны с `--debug` и через `status`.

## NetBird

Посмотрите IP устройства в NetBird dashboard или через `netbird status`; подробный вывод `netbird status -d` показывает peers и FQDN. Используйте NetBird IP, а не публичный адрес или LAN IP. См. [NetBird CLI](https://docs.netbird.io/get-started/cli) и [описание DNS/FQDN](https://docs.netbird.io/about-netbird/how-netbird-works).

`listen.address` — обязательный явный IP. Clipare не выбирает интерфейс автоматически. Убедитесь, что NetBird подключён до запуска и что правила NetBird и локального firewall разрешают TCP 45873 между доверенными peers. Новое устройство добавляется в `peers` всех остальных, затем процессы перезапускаются. Для локальной диагностики допустим `127.0.0.1`.

## API и безопасность

Модель защиты: **NetBird/WireGuard для шифрования канала + HMAC-SHA256 с shared secret для аутентификации Clipare**. Сам HTTP не шифрует данные: не выставляйте endpoint в Интернет или недоверенную LAN. Все владельцы shared secret считаются доверенными и могут подписать запрос от любого настроенного source; это не индивидуальные ключи устройств.

Оба endpoint требуют подпись:

```text
POST /api/v1/clipboard
GET  /api/v1/health
X-Clipare-Timestamp: <Unix seconds>
X-Clipare-Signature: <hex HMAC-SHA256(secret, timestamp + exact raw body)>
```

Для GET тело пустое. Проверка timestamp допускает ±60 секунд; подпись сравнивается constant-time. Часы компьютеров должны быть синхронизированы. Проверить health проще через `clipare status --config config.yaml`: команда подписывает GET и проверяет доступность local endpoint и peers. Обычный неподписанный `curl http://<IP>:45873/api/v1/health` должен вернуть HTTP 401.

Успешный health возвращает `{"status":"ok","device":"macbook-home","mode":"bidirectional"}`. Clipboard endpoint принимает только известный `source` из peers, версию 1, UUIDv4/v7, `text/plain`, UTF-8 без NUL и текст до 1 MiB. Сырой JSON ограничен 6 MiB + 1024 байта с учётом JSON escaping. Timestamp самого сообщения также проверяется. Неизвестные поля и несколько JSON-объектов отклоняются. Ошибки: 400 — сообщение, 401 — подпись, 403 — source, 409 — приём отключён, 413 — размер HTTP body, 503 — clipboard недоступен.

```json
{
  "version": 1,
  "id": "0199f67e-0000-7000-8000-000000000001",
  "source": "macbook-home",
  "timestamp": 1791412345,
  "mime": "text/plain",
  "data": "docker compose ps"
}
```

Это пример формата, а не готовый подписанный запрос. Новый локальный message использует UUIDv7. MIME и версия оставляют возможность расширить протокол позднее.

Повторные ID игнорируются: cache до 1000 ID, TTL 5 минут. Дополнительно SHA-256 текущего содержимого подавляет события от удалённой записи. Для одновременных изменений используется порядок `(timestamp, id)`: все peers выбирают одинакового победителя при доставке обоих сообщений. UUIDv7 локальных изменений монотонно упорядочены даже при быстрых копированиях в одну секунду. Это best-effort синхронизация: при потере сети или различии часов последнее реальное действие пользователя не всегда победит. После TTL возможен повтор старого подписанного health-запроса в допустимом окне; health не меняет состояние.

Содержимое clipboard, тело запросов и secret не логируются. JSON-логи содержат ID, hash, размер, device и peer. Secret хранится только в config; интерфейс `SecretProvider` позволяет позднее подключить Keychain/Credential Manager. Копируемый текст, включая пароли, синхронизируется между всеми настроенными peers: автоматического распознавания секретного содержимого в этой фазе нет.

## Архитектура и зависимости

```text
cmd/clipare            CLI, lifecycle, graceful shutdown
internal/config        строгий YAML и validation
internal/clipboard     WinAPI / NSPasteboard
internal/security      HMAC, timestamp, SecretProvider
internal/sync          сообщения, hash, TTL-cache, serialized clipboard access
internal/transport     HTTP, per-peer workers, health
```

Windows использует `AddClipboardFormatListener` с message-only window. Очередь WinAPI-событий обслуживается каждые 20 мс; сам clipboard не опрашивается. macOS проверяет `NSPasteboard.changeCount` каждые 400 мс. Уведомления объединяются, а текст читается под общей блокировкой с удалённой записью, чтобы отложенное событие не отправило старый snapshot. Память очередей и cache ограничена. Context cancellation завершает watcher и workers, HTTP server получает до 5 секунд на shutdown.

Перед выбором изучены [golang.design/x/clipboard](https://github.com/golang-design/clipboard), его [releases](https://github.com/golang-design/clipboard/releases), [go.mod](https://github.com/golang-design/clipboard/blob/main/go.mod), и [fyne-io/systray](https://github.com/fyne-io/systray). Универсальный clipboard-пакет поддерживает больше форматов и платформ, чем требуется MVP, и имеет несколько зависимостей. Собственные текстовые адаптеры позволяют контролировать лимиты, event loop и shutdown. Systray не добавлен до Phase 2; выбор и актуальные требования CGO будут перепроверены перед её реализацией.

## Проверки и приёмка

```sh
go test -race ./...
go vet ./...
```

Тесты покрывают HMAC и границы timestamp, YAML, UUID/JSON, SHA-256, cache TTL/ёмкость, три устройства и подавление эха, concurrent receive, неуспешную запись, одновременные изменения, отказ endpoint без подписи, лимиты, отсутствие текста в логах, независимость peers и cancellation workers.

Сборка и unit/race-тесты не заменяют системную приёмку. На реальной паре Windows/macOS:

1. Настройте NetBird, full mesh peers и одинаковый secret; запустите оба приложения.
2. Проверьте `status` и отказ health без HMAC.
3. Скопируйте уникальный текст на Mac и вставьте на Windows; затем наоборот. На доступной сети ожидается задержка около секунды.
4. Подключите третий peer, проверьте отсутствие повторных отправок в debug-логах.
5. Отключите один peer и убедитесь, что другие продолжают синхронизацию.
6. Проверьте несколько быстрых копирований, Unicode, пустую строку, размер более 1 MiB и конкурентные копирования.
7. Оставьте процессы работающими на несколько часов, наблюдая память/число потоков; проверьте остановку во время отправки.

Полная межплатформенная приёмка и длительный soak-test пока не выполнены. После их успешного завершения — Phase 2: tray/menu bar, pause/resume, настройки и user-level автозапуск.
