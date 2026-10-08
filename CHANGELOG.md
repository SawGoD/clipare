# Changelog

## 0.5.0

### Нововведения

- Новый Fluent-inspired интерфейс Windows 11: карточки, системная типографика Segoe UI Variable и акцентные основные действия.
- Автоматическое применение системной светлой/тёмной темы и accent color без перезапуска; поддержка Windows Contrast themes.
- Новый список устройств с отдельными состояниями «Подключено» и «Не в сети», обновлённые окна discovery, pairing и обновлений.
- Технические настройки и legacy-коды визуально отделены в «Дополнительно»; SAS-код выделен крупной типографикой.

### Исправления

- Размеры controls и шрифтов учитывают DPI каждого окна, включая перенос между мониторами.
- На небольших экранах содержимое прокручивается; переход Tab доводит скрытый control до видимой области.
- Исправлены фон подписей, иерархия primary/secondary/destructive actions и отображение длинных имён устройств.
- Добавлены пустые состояния списка устройств и discovery, а также действия Enter/Escape для диалогов.

## 0.4.1

### Исправления

- Если установлена последняя версия, окно обновлений показывает текущую версию и кнопку «Понятно» вместо «Позже».
- Состояния проверки, загрузки, готовности и ошибки получили отдельные действия; при ошибке можно повторить проверку или загрузку.
- Кнопка «Позже» отображается только при наличии доступного обновления. Во время загрузки и установки она скрыта.
- Описания GitHub Releases формируются из соответствующего раздела CHANGELOG со ссылкой на сравнение версий.

## 0.4.0

- Add optional daily and manual GitHub Releases checks in native settings and tray menus.
- Download platform updates after explicit confirmation, verify SHA256SUMS, safely extract and validate package/version metadata.
- Apply complete macOS bundles and Windows packages with a temporary helper, graceful shutdown, startup acknowledgement and rollback.
- Preserve installation paths, autostart and device/pairing configuration; suspend new pairing during updates.
- Add bounded HTTPS/redirect policy, streaming downloads and marked staging cleanup. Release-manifest signing remains a documented follow-up; SHA256SUMS alone is not independent publisher authentication.

## 0.3.0

- Discover connected Clipare devices through the local NetBird CLI, without tokens or a cloud service.
- Add committed X25519 pairing with matching six-digit SAS, explicit approval, expiration and replay protection.
- Persist device identities and derive peer-specific HMAC keys without transmitting a group master secret.
- Reconcile authenticated group membership automatically, including third-device onboarding and removal tombstones.
- Simplify native settings with discovered devices and pairing dialogs; move technical fields and legacy codes to Advanced.
- Migrate existing configuration atomically with a private backup and preserve legacy clipboard compatibility.
- Require existing-group confirmation regardless of which device starts pairing; reject conflicting established groups.
- Limit automatic discovery/pairing to the standard NetBird IPv4 range and port; identity re-addition and group merging are not supported yet.

## 0.2.3

- Recognize real Winsock WSAEADDRNOTAVAIL errors during NetBird startup on Windows.
- Verify recovery against a real Windows bind failure, not only synthetic errors.
- Includes the startup recovery changes staged in v0.2.2; that release was cancelled before publication.

## 0.2.2

- Automatically start synchronization when the configured NetBird IP becomes available after login.
- Save valid settings while waiting for NetBird, without repeated error dialogs or manual retries.
- Preserve unsaved peer edits during background recovery; occupied ports remain explicit errors.
- Add GUI recovery integration coverage using a synthetic clipboard and isolated settings.

## 0.2.1

- Copy Windows clipboard memory through WinAPI without integer-to-Go pointer conversion; all platform vet checks remain enabled.
- First published desktop release. The v0.2.0 tag was retained after its CI checks blocked publication.

## 0.2.0

- Native Windows tray and macOS menu bar with peer status, pause/resume and quit.
- Graphical device settings, private config storage and connection-code exchange.
- User-level startup via Windows HKCU Run and macOS LaunchAgent.
- Authenticated peer-to-peer text clipboard sync over NetBird, with bounded queues, UUIDv7 ordering and loop prevention.
- Fixed macOS Command+V and other standard text editing shortcuts.
- Automated multi-platform GitHub Releases and a single version source.
