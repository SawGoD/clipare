# Changelog

## Unreleased — подключение внутри карточки устройств

### Нововведения

- Pairing на Windows и macOS продолжает поиск внутри карточки «Устройства»: код проверки, подтверждение, ожидание и отмена больше не заменяют главный экран.
- Технические поля получили доступные с клавиатуры кнопки «?» с подсказками по наведению и нажатию.

### Исправления

- Убран крупный дублирующий заголовок legacy-блока, уточнены подписи ручных полей другого компьютера.
- Фоновые обновления peers не перекрывают pairing. Возврат корректно отклоняет входящий запрос или отменяет исходящий.
- Windows выбирает действие Enter по состоянию интерфейса, а не по общему контейнеру поиска и pairing.

## 0.5.0-beta.5

### Исправления

- На Windows и macOS кнопки «Применить» для имени компьютера и расширенных параметров скрыты, если связанные настройки не изменены.
- Кнопки появляются при редактировании и исчезают после успешного сохранения либо возврата исходных значений. При ошибке сохранения изменения остаются доступными для повторного применения.
- Раскрытие расширенных параметров само по себе не показывает «Применить»; несохранённое ручное добавление устройств по-прежнему можно применить.

Это **beta для ручного тестирования**, а не stable. Установка вручную; стабильные клиенты не получают prerelease через автоматическое обновление.

## 0.5.0-beta.4

### Нововведения

- Поиск и добавление устройств Windows/macOS размещены внутри карточки устройств главного экрана. Статус, имя компьютера и настройки остаются видимыми.
- Высота списка найденных устройств адаптируется к числу результатов и ограничена для длинных списков.

### Исправления

- Возврат восстанавливает обычный список устройств; переход к ручному подключению отменяет discovery.
- Фоновое обновление статуса peers не перекрывает поиск; Windows refresh сохраняет фокус внутри текущего блока.

Это **beta для ручного тестирования**, а не stable. Установка вручную; стабильные клиенты не получают prerelease через автоматическое обновление.

## 0.5.0-beta.3

### Исправления

- При запуске без устройств таблица и компактный «+» скрыты сразу, даже если первый refresh пропускается кэшем.
- Пустой блок ограничен по высоте и не растягивается при увеличении окна; остаётся только центральная кнопка добавления.
- Нативный тест воспроизводит реальный старт без принудительного refresh и проверяет геометрию Light/Dark, изменение размера и удаление последнего устройства.

Это **beta для ручного тестирования**, а не stable. Установка вручную; стабильные клиенты не получают prerelease через автоматическое обновление.

## 0.5.0-beta.2

### Нововведения

- Версия и иконка проверки обновлений перенесены в компактную нижнюю строку без отдельной карточки.
- Технические поля размещены в две колонки; корзина с подтверждением доступна у каждого устройства.
- Связанные с пустыми полями действия отключены; ошибки показываются небольшим системным диалогом, не заменяя текущий экран.

### Исправления

- Удаление устройства больше не проверяет незаполненную техническую форму и сохраняет остальные настройки.
- Применение имени компьютера не отправляет всю форму расширенных параметров.
- Убрана синхронная Windows-перерисовка при движении мыши и лишние обновления неизменившегося статуса.
- macOS не пересобирает список и меню при одинаковых данных; смена appearance не пересоздаёт material hierarchy.
- Корзины не обрезаются на macOS 15 и Tahoe благодаря явному стилю таблицы и привязке к границе строки.

Иконки конкретной ОС пока не отображаются: текущая metadata доверенных устройств не содержит достоверной платформы. Определение ОС по hostname не используется. Реальная приёмка плавности Windows 11/macOS и межплатформенной синхронизации остаётся необходимой.

Это **beta для ручного тестирования**, а не stable. Установка вручную: стабильные клиенты не получают prerelease через автоматическое обновление.

## 0.5.0-beta.1

### Нововведения

- Тестовый prerelease нового desktop-интерфейса: Windows Fluent-inspired оформление и macOS Liquid Glass с нативным material fallback.
- Одно основное окно для discovery, pairing, обновлений, ошибок и раскрываемых расширенных параметров.
- Единый статус синхронизации в окне и tray/menu bar: работает, приостановлено, недоступно; выключенный peer не меняет глобальный статус.
- Empty state с центральной кнопкой добавления, компактный список устройств и системные иконки действий.
- Системные Light/Dark и accent color, адаптация macOS к Reduce Transparency, мгновенное применение флажков автозапуска и обновлений.

### Исправления

- Улучшены Windows-шрифты, галочки, поля ввода, выпадающий список, DPI и прокрутка вложенных экранов.
- При отсутствии обновлений отображаются текущая версия и «Понятно» вместо «Позже».
- Inline-ошибки не теряют текущий pairing; предложение обновления не перекрывает подключение устройства.
- Цвета непрозрачных macOS-поверхностей учитывают тему конкретного окна; сохранены Cmd+Z/X/C/V/A.
- Отклонённые флажки возвращаются к актуальным значениям, завершённый pairing не создаёт запоздалых уведомлений.

Это **beta для ручного тестирования**, а не stable. Реальная визуальная приёмка macOS 26/Windows 11 и межплатформенные регрессии ещё не завершены. Сборки прошли unit/race/vet, нативные UI fixtures и CI. Установка вручную; stable-клиенты не получают эту beta через автообновления. Beta-сборка также не устанавливает stable-обновления автоматически.

## Unreleased — общий Desktop UX (Release A)

### Нововведения

- Discovery, pairing, обновления и обычные сообщения используют одно основное окно Windows/macOS.
- Расширенные технические параметры раскрываются inline и не повторяют обычные настройки.
- Общий статус active/disabled/degraded отображается в окне, tray/menu bar и tooltip; offline peer не меняет статус утилиты.
- Системные иконки действий с tooltip/accessibility labels, мгновенное применение простых флажков.

### Исправления

- Сообщения внутри окна не теряют текущий pairing workflow; предложение обновления не перекрывает pairing.
- Убраны остаточные scrollbars при переходах между длинными и короткими Windows-панелями.
- Отклонённые/неуспешные изменения флажков возвращаются к актуальным значениям.

## Unreleased — macOS Glass (Release B, после A)

### Нововведения

- Компактные AppKit-карточки с NSGlassEffectView на macOS 26 и NSVisualEffectView fallback на старой ОС; Glass доступен и при сборке старым SDK.
- Системные accent color, SF Symbols, прозрачный title bar и адаптивные поверхности.
- Reduce Transparency переключает карточки на непрозрачные системные цвета без перезапуска.

### Исправления

- Цвета слоёв вычисляются в appearance конкретного окна, сохраняя контраст Light/Dark.
- Menu bar mark адаптируется к теме, сохраняя цветной status badge.
- Старое разрешение notification не создаёт уведомление для завершённого pairing.
- Сохранены Command+Z/X/C/V/A, закрытие окна и навигация с клавиатуры.

## 0.5.0

### Нововведения

- Пустой блок устройств показывает центральную круглую кнопку добавления; действия списка появляются только после подключения устройства.
- Автозапуск и автоматическая проверка обновлений объединены в сворачиваемую секцию «Дополнительно»; технические настройки переименованы в «Расширенные параметры».
- Новый Fluent-inspired интерфейс Windows 11: карточки, системная типографика Segoe UI и акцентные основные действия.
- Автоматическое применение системной светлой/тёмной темы и accent color без перезапуска; поддержка Windows Contrast themes.
- Новый список устройств с отдельными состояниями «Подключено» и «Не в сети», обновлённые окна discovery, pairing и обновлений.
- Технические настройки и legacy-коды визуально отделены в «Дополнительно»; SAS-код выделен крупной типографикой.

### Исправления

- Поле выбора NetBird и раскрытый список используют цвета текущей темы, современную рамку и выделение; Enter/Escape сохраняют нативное поведение раскрытого списка.
- Убрано искусственное утолщение заголовков, включён Natural ClearType; обновлены галочки, рамки полей и стрелка выпадающего списка.
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
