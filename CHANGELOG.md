# Changelog

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
