# Clipare

## Windows-интерфейс 0.5.0

Windows использует лёгкий Win32-интерфейс в направлении Fluent / Windows 11: карточки с радиусами 6–8 px, Segoe UI с Natural ClearType, системный accent и Light/Dark. Для крупных заголовков используется отдельное Semibold-начертание, без искусственного утолщения текста. Галочки, рамки полей и стрелка списка оформлены в общей стилистике. Тема обновляется при изменении системных настроек, Contrast themes сохраняют системные цвета. HWND controls, стандартное редактирование и навигация остаются нативными; Electron, WebView и новый UI runtime не добавлены. Рамка окна использует поддерживаемые DWM dark mode и rounded corners; клиентская область намеренно непрозрачна — это безопасный fallback, а не имитация Mica ручным blur.

Основное окно показывает текущее устройство, список устройств и обновления. «Дополнительно» раскрывает флажки автозапуска и обновлений, а «Расширенные параметры» — ID, IP, порт и legacy-коды. Online/offline не вызывают уведомления; недоступные устройства не выделяются красным. При переносе окна между мониторами обновляются размеры и шрифты; если рабочая область мала, включается прокрутка. Tab/Shift+Tab показывают control с фокусом, Enter выполняет выбранное действие, Escape возвращает к основному экрану (или отклоняет/отменяет pairing).

В beta.9 клики мышью сохраняют спокойные рамки полей; клавиатурная навигация показывает индикатор фокуса. Прокрутка использует системную буферизацию дочерних controls и пакетное изменение позиций, без повторного назначения шрифтов на каждом шаге. Пустой поиск, сканирование и ошибка NetBird показывают только одно сообщение состояния на обеих платформах.

### Проверка Windows redesign перед публикацией

Workflow `Windows UI preview` запускается вручную и публикует **тестовые artifacts, не GitHub Release**: x64/ARM64 ZIP и синтетические native screenshots. Fixture не читает clipboard/config, не запускает NetBird и не меняет autostart или системную тему. Light/Dark/accent в снимках — тестовые палитры, не замена ручной приёмки.

Перед публикацией проверить на настоящей Windows 11: Light/Dark, несколько системных accent colors, 100/125/150/200% scaling и перенос окна между мониторами; Tab/Shift+Tab/Enter/Escape, длинные имена, пустые списки, discovery и SAS. Отдельно проверить состояния обновлений и существующие Mac ↔ Windows sync, pairing, membership, autostart и сохранение настроек.

## Desktop UX beta 0.5.0-beta.1

Beta объединяет новые интерфейсы для ручной проверки: одно окно для discovery/pairing/обновлений/ошибок, inline Advanced и macOS Glass. Это не stable release: визуальная приёмка и межплатформенные регрессии ещё не завершены. GitHub Release помечается Prerelease и не заменяет Latest stable. Beta устанавливается вручную; обычный updater игнорирует prereleases, а beta/dev-сборки не устанавливают stable-обновления автоматически. Stable-этапы общего UX и macOS-оформления по-прежнему должны выйти раздельно после приёмки.

Общий статус отображается в окне, меню и иконке: зелёный — session работает; красный — пользователь приостановил sync; оранжевый — sync включён, но listener/session недоступны. Выключенный peer не меняет глобальный статус. Цвет всегда сопровождается текстом и tooltip.

macOS использует AppKit, системные шрифты, SF Symbols и accent color. На macOS 26 карточки используют публичный [NSGlassEffectView](https://developer.apple.com/documentation/appkit/nsglasseffectview), иначе — NSVisualEffectView. Runtime-проверка класса/публичных selectors позволяет и сборкам старого SDK использовать Glass на Tahoe. При Reduce Transparency карточки переходят на непрозрачные системные цвета. Смена appearance и accessibility settings применяется без перезапуска; минимальная версия ОС не повышена.

`make test-macos-ui` проверяет один NSWindow, empty/list states, inline sections, pairing, обновления, Light/Dark цвета непрозрачного режима, material fallback и Cmd+V. Fixture не читает clipboard/config, не генерирует ключи и не меняет системные настройки. Workflow `macOS UI preview` публикует Intel/Apple Silicon ZIP и диагностические снимки. Bitmap AppKit не воспроизводит GPU Glass полностью, поэтому эти снимки **не заменяют** визуальную проверку на настоящей macOS 26.

Чек-лист и границы отдельных релизов: [desktop-ux-validation.md](docs/desktop-ux-validation.md).

Минималистичная peer-to-peer синхронизация текстового буфера обмена через уже настроенную сеть NetBird. Go, без центрального сервера, аккаунтов, истории clipboard и браузерного UI.

Реализованы headless-синхронизация, нативный tray/menu bar для Windows 11 и macOS, автоматическое обнаружение устройств через локальный NetBird CLI и подключение с проверочным кодом. Обычный запуск открывает настройки только при первом старте; затем приложение работает в tray без постоянного окна. Мобильные устройства, картинки, файлы и HTML не поддерживаются.

## Быстрый старт через GUI

Редактировать YAML не требуется.

1. На Mac соберите `make mac` и откройте `dist/Clipare.app`; на Windows запустите `Clipare.exe` (сборка ниже).
2. Убедитесь, что оба компьютера подключены к NetBird. Clipare автоматически выбирает доступный NetBird IP и создаёт постоянную X25519 identity; имя можно изменить в обычном окне.
3. Нажмите «+ Добавить устройство». Clipare получает `netbird status --json`, проверяет online peers на порту 45873 и показывает только доступные Clipare-клиенты, ещё не добавленные в группу.
4. Выберите второй компьютер и нажмите «Подключить». Сверьте шестизначный код на обоих экранах, затем нажмите «Разрешить» на принимающем компьютере. Если подключение начинает уже существующий member, на нём также нажмите «Код совпадает»: добавление требует подтверждения существующей группы. Если коды отличаются, отклоните или отмените подключение. Адреса, ID и ключи сохраняются автоматически; обратное ручное добавление не требуется.
5. В меню Clipare отображаются `●` online / `○` offline. Проверка выполняется раз в 20 секунд. Попробуйте копирование текста в обе стороны.

Для третьего устройства выполните pairing только с одним существующим member. Metadata полного mesh распространяется сразу после изменения и затем сверяется раз в 20 секунд; offline peers догоняют состав после восстановления сети. Объединение двух уже существующих групп не выполняется автоматически.

Если NetBird CLI отсутствует или недоступен, приложение покажет «Автоматическое обнаружение недоступно». В «Дополнительно…» остаются ручные поля и «Добавить по коду»: приватно передайте `clipare1:…`, импортируйте и сохраните его, затем добавьте обратный код на другом компьютере. Для legacy full mesh коды по-прежнему нужны между всеми парами. Код содержит legacy shared secret, но никогда не содержит private identity key. Clipare не отправляет `clipare1:…` через автоматическую синхронизацию clipboard.

При обновлении старого config peers и legacy secret сохраняются. Перед атомарной миграцией в schema 2 создаётся `config.yaml.v1.backup` с правами 0600 на Unix. Уже доверенные обновлённые клиенты аутентифицированно обмениваются public keys и переходят на pairwise keys; старые версии продолжают работать в legacy-режиме. Для автоматического mesh все его участники должны поддерживать новый membership API. Приватные ключи и секреты находятся только в пользовательском storage; не публикуйте config или backup.

Discovery/pairing поддерживает стандартный NetBird IPv4 диапазон `100.64.0.0/10` и Clipare port 45873. Для нестандартного порта используйте advanced/manual flow; legacy traffic в явно настроенных приватных диапазонах сохраняется, но автоматический pairing в них пока не поддерживается. При наличии FQDN сохраняются hostname и последний известный NetBird IP; запросы проверяют разрешённый IP, а при отказе FQDN используется IP fallback.

### Повторное добавление удалённого устройства

Начиная с **0.5.0-beta.8**, на компьютере, где устройство удалено, откройте «Добавить устройство», выберите его в поиске и выполните новый pairing: сверьте код и явно подтвердите подключение. Сброс config, Device ID и ключей не требуется. Не удаляйте настройки вручную: они содержат identity и связи с другими компьютерами.

Перед повторным добавлением обновите **все компьютеры группы до beta.8 или новее**. Версии изменений состава группы позволяют отличать новое подтверждённое подключение от запоздавшего удаления. Старые версии не понимают новые `membership_versions` в config и `versions` в membership API; откат к ним после сохранения этого состояния не поддерживается. Clipboard-протокол и legacy `clipare1:` не изменились. Новый pairing не объединяет две разные уже существующие группы.

«Приостановить синхронизацию» в tray отключает отправку и приём; «Возобновить» включает их. «Открыть Clipare» открывает окно, крестик скрывает его, **«Выйти» завершает процесс**. На Windows иконка может оказаться в списке скрытых значков рядом с часами; на Mac используется clipboard-иконка с цветным статусом в menu bar.

Windows: повторный запуск GUI с тем же config открывает уже работающий экземпляр; второй listener не запускается. Если первый процесс не отвечает, выводится понятная ошибка, вместо запуска конкурирующей копии. При первой установке beta.8 завершите старый Clipare через «Выйти» (или диспетчер задач, если он завис): старые версии ещё не поддерживают активацию существующего экземпляра. `--headless` и `status` остаются отдельными диагностическими режимами.

Флажок «Запускать при входе в систему» применяется сразу: Windows — HKCU Run, macOS — `~/Library/LaunchAgents/io.clipare.agent.plist`. Администратор не нужен. Автозапуск использует абсолютный путь к программе: разместите её в постоянном месте перед включением, а после перемещения выключите и снова включите флажок. Если Clipare стартовал раньше NetBird, он покажет «Ожидание NetBird» и автоматически повторит запуск примерно через 5–7 секунд. Настройки сохраняются и при временно отсутствующем локальном IP. Ошибка занятого порта показывается отдельно — проверьте, не запущен ли второй экземпляр.

Настройки автоматически сохраняются в пользовательском каталоге:

- macOS: `~/Library/Application Support/Clipare/config.yaml` (права 0600).
- Windows: `%AppData%\Clipare\config.yaml` (доступ наследуется от пользовательского каталога).

Старый config можно открыть через `--config /path/config.yaml`: UI позволяет его изменить. Автоматически читать локальные конфиги проекта или переносить их в пользовательский каталог приложение не будет.

## Сборка

Нужен Go 1.23+. Единственная внешняя runtime-зависимость Go — `go.yaml.in/yaml/v3 v3.0.5` для конфигурации: [форк официальной YAML-организации](https://github.com/yaml/go-yaml). Линия v3 получает security fixes, сохраняя API; CGO и системных зависимостей у парсера нет. Исходный `gopkg.in/yaml.v3` архивирован и заменён.

macOS (arm64 или amd64): установите Xcode Command Line Tools (`xcode-select --install`). Адаптер NSPasteboard написан на Objective-C, поэтому CGO включён; используется системный AppKit. Дополнительный runtime не требуется.

```sh
go build -o clipare ./cmd/clipare
```

Для запуска двойным кликом с корректным menu bar и без Dock-иконки соберите `.app`:

```sh
make mac
```

Windows 11 (PowerShell, CGO не нужен):

```powershell
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags "-H=windowsgui" -o Clipare.exe ./cmd/clipare
```

Кросс-компиляция Windows с Mac:

```sh
make windows
```

Для Windows arm64 замените `GOARCH` на `arm64`. Сборку macOS выполняйте на macOS с соответствующим SDK. Linux не входит в поддерживаемые платформы; протокол и тесты могут запускаться без native clipboard.

## Версии и GitHub Releases

Готовые архивы публикуются в [GitHub Releases](https://github.com/SawGoD/clipare/releases). Для Mac выбирайте `darwin-arm64` (Apple Silicon) или `darwin-amd64` (Intel), для Windows — `windows-amd64` или `windows-arm64`. Архив Mac содержит `Clipare.app`; Windows — GUI `Clipare.exe` и диагностический `clipare-console.exe`. `SHA256SUMS` позволяет проверить загрузку.

Единственный источник номера версии — файл `VERSION`, формат SemVer `MAJOR.MINOR.PATCH`. Версия приложения и `Info.plist` берутся из него; release-бинарники также содержат Git commit (`--version`). Пока поддерживаются стабильные теги `vX.Y.Z`; тег обязан совпадать с `VERSION`.

Для следующего релиза:

1. Обновите `VERSION` и `CHANGELOG.md`: PATCH для исправлений, MINOR для новых возможностей, MAJOR для несовместимых изменений. До 1.0 несовместимые изменения отмечайте новой MINOR-версией.
2. Выполните `make test`, проверьте diff и создайте подписанный Conventional Commit.
3. Создайте подписанный тег и отправьте его вместе с веткой:

```sh
git tag -s v0.3.1 -m "Clipare v0.3.1"
git push origin master
git push origin v0.3.1
```

Workflow запускает проверки на Linux/macOS/Windows. После успешных проверок он собирает четыре архива, проверяет SHA-256 и публикует Release с автоматически сформированными notes. Обычные push/PR запускают только CI; автоматического повышения версии при каждом коммите нет. Неправильный тег или неуспешная сборка блокируют публикацию. Существующий Release не перезаписывается.

Локальная проверка release-сборок:

```sh
make release-mac       # на macOS: Apple Silicon + Intel
make release-windows   # Windows amd64 + arm64; можно собирать на Mac/Linux
```

Mac release имеет ad-hoc подпись, но не Apple Developer ID и notarization; Windows не имеет Authenticode подписи. При загрузке ОС может показывать предупреждение о неизвестном разработчике. Подписи Git-коммитов и SHA-256 не заменяют системную подпись приложения. Apple/Microsoft сертификаты и ключи в этот процесс не добавлялись.

## Автоматические обновления

Начиная с 0.4.0, в обычных настройках есть блок «Обновления», текущая версия и флажок «Автоматически проверять обновления» (по умолчанию включён). Выключение флажка сохраняется сразу; в YAML это `updates.enabled: false`. Ручная «Проверить обновления» доступна также в tray/menu bar и не зависит от флажка или интервала. Headless-режим не проверяет и не устанавливает обновления.

Фоновая проверка использует официальный public GitHub Releases API репозитория SawGoD/clipare без token, облачного update-сервера и обращения к NetBird. Попытки проверок сохраняются в отдельном `state.json`; повторная автоматическая попытка выполняется не чаще раза в 24 часа, в том числе после ошибки или перезапуска. Никакие ключи, peers или clipboard в GitHub не отправляются. Учитываются только stable SemVer `vX.Y.Z`, не draft/prerelease; более старые версии не предлагаются. Dev/prerelease-сборки не обновляются автоматически и не устанавливают stable-пакеты через этот механизм.

Новая версия предлагает «Обновить» / «Позже». **Установка всегда требует нажатия «Обновить».** После него загрузка и установка идут в фоне: нужный ZIP → SHA256SUMS → безопасная распаковка → проверка platform/version metadata и `--version` нового binary → временный helper → штатное завершение sync/pairing → замена пакета → запуск новой версии. На Mac дополнительно проверяется целостность ad-hoc bundle подписи через `codesign --verify`; quarantine/Gatekeeper не отключаются. При активном pairing установка заблокирована. Новые pairing requests при подготовке обновления временно не принимаются.

Обновляется весь `Clipare.app` на Mac, весь список package-файлов на Windows (включая GUI и console binary). Папка установки и абсолютные пути autostart сохраняются. Пользовательский config, identity, peers, tombstones и настройки не входят в update package и не заменяются. Неизвестные файлы рядом с Windows binary сохраняются. Устанавливать Clipare нужно в доступную текущему пользователю папку; updater не вызывает sudo/UAC. Отдельный Mac executable вне `Clipare.app` и Windows console-only запуск не поддерживают self-update. При недоступной для записи `/Applications` переместите bundle в `~/Applications` либо обновите вручную.

До успешного staging старая установка не затрагивается. Helper создаёт sibling transaction directory, сохраняет предыдущие файлы, заменяет их и ждёт подтверждения запуска GUI с успешно прочитанным config. При ошибке замены или запуска восстанавливает старые файлы и запускает предыдущую версию. Это проверка старта, не проверка clipboard/NetBird после обновления. При аппаратной ошибке, препятствующей самому rollback, единственная резервная копия сохраняется в `.clipare-update-*/backup` рядом с установкой; не удаляйте её. Аварийное выключение питания посреди нескольких Windows rename не обеспечивает ACID-транзакцию: сохранённые backup позволяют восстановить пакет вручную.

Временные файлы находятся в `os.UserCacheDir()/Clipare/update`: `~/Library/Caches/Clipare/update` на Mac, `%LocalAppData%/Clipare/update` на Windows. Успешные staging-каталоги удаляются при фоновой housekeeping-проверке спустя минуту, заброшенные — спустя семь дней. Очищаются только помеченные updater-каталоги, не config и не резервные копии установки. HTTP metadata ограничена 1 MiB / 10 секунд, ZIP — 128 MiB / 5 минут, распакованные файлы — 512 MiB. Загрузка потоковая; только HTTPS и точный список GitHub/CDN hosts, включая `release-assets.githubusercontent.com`, без произвольных redirects. Updater использует отдельный HTTP client с обычной Go proxy policy (environment), peer traffic по-прежнему не использует proxy.

**Ограничение доверия:** SHA256SUMS из того же Release подтверждает целостность ZIP, но не является независимым доказательством происхождения. Сейчас доверие основано на HTTPS и безопасности GitHub repository/release publishing. Встроен отдельный `ManifestVerifier` и Ed25519 verifier для будущего обязательного signed manifest. Dedicated public release key ещё не закреплён; private signing secret в GitHub Actions не создавался. Device X25519 keys не используются для обновлений. До внедрения release signing компрометация аккаунта/pipeline GitHub может привести к установке подменённого пакета. Проверяйте, кому разрешены release tags и публикация assets.

Архивы 0.4.0+ содержат `update-package.json`, покрытый SHA256SUMS. Релизы до 0.4.0 не имеют updater; переход с 0.3.x на 0.4.0 нужно выполнить один раз вручную. Подпись manifest и реальная проверка обновления между двумя опубликованными версиями — отдельные этапы приёмки.

## Настройка вручную и CLI

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
./clipare --headless --config /absolute/path/config.yaml
./clipare --headless --config /absolute/path/config.yaml --debug
./clipare status --config /absolute/path/config.yaml
./clipare --version
```

Для терминальной диагностики на Windows соберите без `-H=windowsgui` либо выполните `make windows-console`; GUI-бинарник не имеет консоли для вывода `status`/логов. Приложение работает в интерактивном пользовательском сеансе, где доступен clipboard. Начальное содержимое при запуске или применении настроек не отправляется: синхронизация начинается с нового копирования. Для остановки headless — Ctrl+C; на macOS также SIGTERM. Изменения YAML в headless требуют перезапуска; через GUI изменения применяются сразу.

`sync.enabled: false` либо `sync.mode: disabled` отключает отправку и приём; health остаётся доступным. Также реализованы `send-only` и `receive-only`; по умолчанию `bidirectional`. Отправитель делает одну попытку доставки, timeout 2 секунды. Каждый peer имеет свой worker и очередь на одно последнее ожидающее сообщение. Health проверяется раз в 20 секунд, статусы видны с `--debug` и через `status`.

## NetBird

Посмотрите IP устройства в NetBird dashboard или через `netbird status`; подробный вывод `netbird status -d` показывает peers и FQDN. Используйте NetBird IP, а не публичный адрес или LAN IP. См. [NetBird CLI](https://docs.netbird.io/get-started/cli) и [описание DNS/FQDN](https://docs.netbird.io/about-netbird/how-netbird-works).

`listen.address` — обязательный явный IP. При первом GUI-запуске Clipare предпочитает локальный IP из `netbird status --json`, если он присутствует на интерфейсах; при недоступном CLI предлагает CGNAT-адрес как подсказку. Убедитесь, что NetBird подключён и что правила NetBird и локального firewall разрешают TCP 45873 между доверенными peers. Автоматический pairing сам распространяет новых участников; ручное добавление на всех устройствах требуется только в legacy-сценарии. Для локальной диагностики допустим `127.0.0.1`.

## API и безопасность

Модель защиты: **NetBird/WireGuard для шифрования канала + X25519 identity и pairwise HMAC-SHA256 для аутентификации Clipare**. Сам HTTP не шифрует данные: не выставляйте endpoint в Интернет или недоверенную LAN. После SAS-approved pairing постоянный ключ конкретной пары выводится из X25519 identity, group ID и обоих device ID. Третье устройство не получает этот ключ. Новые requests криптографически связаны с source, методом и route; peer A не может подписывать clipboard от имени B. Legacy peers сохраняют прежний shared-secret wire format и его ограничение: владелец общего legacy secret может представляться другим legacy member.

Оба endpoint требуют подпись:

```text
POST /api/v1/clipboard
GET  /api/v1/health
X-Clipare-Timestamp: <Unix seconds>
X-Clipare-Source: <sender device ID, required for paired peers>
X-Clipare-Signature: <hex HMAC-SHA256(pairwise key, timestamp + method + LF + route + LF + source + LF + raw body)>
```

Для legacy подпись остаётся `HMAC-SHA256(secret, timestamp + exact raw body)`. Для GET тело пустое. Проверка timestamp допускает ±60 секунд; подпись сравнивается constant-time. Часы компьютеров должны быть синхронизированы. Проверить health проще через `clipare status --config config.yaml`: команда использует ключ каждого peer. Локальная диагностика допускает config secret только с собственного listen IP; remote legacy health допускается, пока есть legacy relationships. Неподписанный health возвращает HTTP 401.

Иконка Windows/macOS в списке устройств выбирается только по сведениям из аутентифицированного health-ответа конкретного paired peer. Клиент отправляет случайный 256-битный `X-Clipare-Nonce`; ответ содержит optional `platform` (`windows` / `darwin`) и `X-Clipare-Response-Signature` — HMAC на том же timestamp и pairwise key, с данными `RESPONSE + LF + /api/v1/health + LF + device ID + LF + nonce + LF + raw response body`. Nonce связывает ответ с конкретной проверкой и предотвращает replay. ОС заявляет само доверенное устройство: это не аппаратная аттестация. По hostname платформа не угадывается. До pairing, для старых/legacy клиентов и при неподтверждённых данных используется нейтральная иконка компьютера. Metadata кэшируется только в памяти; clipboard, handshake и пользовательский config не изменяются.

`GET /api/v1/discovery` открыт без HMAC и возвращает только protocol/app version, device ID/name, public key и pairing availability. Pairing использует commit/reveal, ephemeral и static X25519, независимое вычисление SAS и обязательное локальное approve. Один активный входящий pairing, срок до 120 секунд, одноразовые sessions, bounded cache, 1 request/s на source IP и лимиты body. Membership API допускает только известного отправителя с pairwise HMAC; tombstones сохраняют удаления. Подробности: [pairing protocol](docs/PAIRING.md).

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

Содержимое clipboard, тело requests, private keys, session keys и legacy secrets не логируются. JSON-логи содержат ID, hash, размер, device и peer. `identity.Provider` и `PeerKeyProvider` отделяют криптографию от config storage для будущего Keychain/Credential Manager. Копируемый текст, включая пароли, синхронизируется между всеми настроенными peers: автоматического распознавания секретного содержимого нет.

## Архитектура и зависимости

```text
cmd/clipare            CLI, lifecycle, graceful shutdown
internal/config        строгий YAML и validation
internal/clipboard     WinAPI / NSPasteboard
internal/security      HMAC, timestamp, SecretProvider
internal/identity      persistent X25519, key derivation, fingerprint
internal/discovery     injectable NetBird CLI, bounded HTTP probes
internal/pairing       committed handshake, SAS, approval, expiry, control API
internal/peers         public membership metadata and revocation tombstones
internal/sync          сообщения, hash, TTL-cache, serialized clipboard access
internal/transport     HTTP, per-peer workers, health
internal/app           restartable sync session, lifecycle
internal/ui            native AppKit / Win32 controls and tray
internal/autostart     LaunchAgent / HKCU Run
```

Windows использует `AddClipboardFormatListener` с message-only window. Очередь WinAPI-событий обслуживается каждые 20 мс; сам clipboard не опрашивается. macOS проверяет `NSPasteboard.changeCount` каждые 400 мс. Уведомления объединяются, а текст читается под общей блокировкой с удалённой записью, чтобы отложенное событие не отправило старый snapshot. Память очередей и cache ограничена. Context cancellation завершает watcher и workers, HTTP server получает до 5 секунд на shutdown.

Перед выбором изучены [golang.design/x/clipboard](https://github.com/golang-design/clipboard), его [releases](https://github.com/golang-design/clipboard/releases), [go.mod](https://github.com/golang-design/clipboard/blob/main/go.mod), и [fyne-io/systray](https://github.com/fyne-io/systray). Универсальный clipboard-пакет поддерживает больше форматов и платформ, чем требуется MVP, и имеет несколько зависимостей. Собственные текстовые адаптеры позволяют контролировать лимиты, event loop и shutdown. Для GUI также используются прямые системные API: AppKit через CGO на Mac, Win32 без CGO на Windows. Новых Go-зависимостей для tray/settings не добавлено; браузерного frontend и GUI runtime нет.

## Проверки и приёмка

```sh
go test -race ./...
go vet ./...
```

Тесты покрывают HMAC и границы timestamp, YAML/атомарную миграцию, UUID/JSON, SHA-256, cache TTL/ёмкость, три устройства и подавление эха, concurrent receive, неуспешную запись, одновременные изменения, отказ endpoint без подписи, лимиты, отсутствие текста в логах, независимость peers и cancellation workers. Новые проверки включают fake NetBird CLI/probes, X25519/SAS, expiration/replay, обязательный approve, обе стороны инициирования pairing, автоматический mesh трёх клиентов, revocation, impersonation и legacy upgrade. На Mac `make test-macos-ui` проверяет нативные окна и Command+V без доступа к реальному clipboard.

Сборка и unit/race-тесты не заменяют системную приёмку. На реальной паре Windows/macOS:

1. Настройте NetBird, запустите оба GUI, найдите друг друга через «Добавить устройство».
2. Проверьте `status` и отказ health без HMAC.
3. Проверьте одинаковый SAS, reject и повторный pairing с approve. Скопируйте уникальный текст на Mac и вставьте на Windows; затем наоборот. На доступной сети ожидается задержка около секунды.
4. Подключите третий peer только через один existing member; убедитесь, что все знают остальных, и проверьте отсутствие повторных отправок.
5. Отключите один peer и убедитесь, что другие продолжают синхронизацию.
6. Проверьте несколько быстрых копирований, Unicode, пустую строку, размер более 1 MiB и конкурентные копирования.
7. Оставьте процессы работающими на несколько часов, наблюдая память/число потоков; проверьте остановку во время отправки.

Предыдущий GUI macOS проходил визуальную проверку; новая Glass-версия пока проверена только нативными fixtures и диагностическими bitmap-снимками. Unit/race-тесты покрывают сохранение настроек, обмен кодами, запрет пересылки кода, повторный запуск сессии и shutdown при отказе watcher. Реальная приёмка нового UI на Windows/macOS, проверка автозапуска после входа и длительный soak-test пока не выполнены.
