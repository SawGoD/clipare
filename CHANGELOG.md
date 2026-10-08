# Changelog

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
