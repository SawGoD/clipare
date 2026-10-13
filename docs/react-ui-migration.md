# React desktop UI migration

## Baseline audit (master, v0.6.0-beta.4)

Clipare currently has no WebView. `internal/ui/ui.go` owns the GUI application
loop and talks to a `desktop` adapter through an event queue, a form and typed
status helpers. Windows is a CGO-free Win32/HWND implementation in
`internal/ui/windows*.go`. macOS uses AppKit/Objective-C through
`internal/ui/darwin.go` and `native_darwin.m`. The same loop owns clipboard
session start/stop, NetBird discovery, pairing, encrypted local history,
autostart preference, GitHub updates and graceful update handoff. Headless mode
in `cmd/clipare/main.go` bypasses `internal/ui` at runtime.

Current views and states: home (zero/one/many devices, online/offline, sync
active/paused/degraded), discovery (scanning/empty/error/results), pairing
(incoming approval, outgoing SAS, waiting, rejection/timeout), history
(empty/grouped/secret masked/revealed/expanded/key error), update
(checking/current/available/downloading/ready/error), inline additional and
technical settings, inline notices. The Windows tray and macOS status item
can reopen the window after closing it. Dialogs also include removal and
history-clear confirmations. Preserve every operation, keyboard shortcut and
error state when replacing the adapters.

`Makefile` and `scripts/build` currently produce four archives with the
existing executable names; `update` validates package metadata and replaces
the complete macOS bundle or Windows file set. `.github/workflows/release.yml`
tests three OSes, then packages four targets. None of these paths may start a
frontend server in production or rewrite user config/history.

## Host choice and migration boundary

The native window hosts cannot show React without embedding WebView2 and
WKWebView plus writing two new JavaScript bridges. That would duplicate window,
event and accessibility infrastructure. The candidate host is a pinned Wails
v3 desktop beta: it supports Windows amd64/arm64 without CGO, WKWebView on
macOS, native tray/window lifecycle, bundled assets and restricted Go service
bindings. [Wails' status page](https://v3.wails.io/status/) still calls v3
beta. Keep the current adapters and release build as rollback until the new
host passes device acceptance. `--headless` must continue to bypass the host.

The principal risk is moving from the current `Poll()` loop and native main
thread into Wails' event loop while retaining single-instance activation,
tray controls and update handoff. Treat this as a separate integration gate;
browser previews alone cannot demonstrate it. Pin dependencies; do not expose
config files, keys, shell commands or network protocol through the React API.

## Frontend layers

- `frontend/src/domain`: typed snapshots, commands and backend contract.
- `frontend/src/preview`: deterministic mock backend, browser-only scenario
  selection and fixture data. URL platform override exists only in Vite dev.
- `frontend/src/platform/windows`: Fluent UI React v9 layout and icons.
- `frontend/src/platform/macos`: separate macOS layout and system-like controls.
- Production adapter: allowlisted Wails service methods and change events;
  backend remains authoritative. Subscription teardown is mandatory.

The browser preview is the development surface for both designs on macOS.
Production platform comes from the desktop host, never the URL. Browser
snapshots use fixed viewport, pixel ratio and mock data. Real Windows 11 and
macOS checks remain release gates, including memory/CPU comparison to the
native baseline.

## Integration in this branch

`internal/ui/react_contract.go` is the allowlisted Wails bridge. It exposes a
typed snapshot and commands, then queues GUI actions into the existing Go
loop. No raw configuration, private identity key or legacy secret is sent to
React. History links are looked up by ID in the current visible snapshot and
only HTTP(S) URLs are opened. All sync, pairing, history encryption and update
installation remain Go-owned. The browser frontend subscribes to state events
and releases the subscription when unmounted; it does not poll continuously.

`internal/ui/react_host_desktop.go` owns the WebView window and native tray.
Closing the window hides it while the Go loop keeps running; Quit cancels the
loop and releases resources. The packaged UI is selected automatically when
`ui/index.html` exists beside Windows binaries or in the macOS bundle
Resources directory. `--native-ui` retains the previous implementation as a
rollback path; direct Go builds without UI assets use it as well. Windows
second-instance activation uses the existing named-instance handshake and a
window subclass. This needs physical Windows validation.

The release builder copies hashed frontend assets into the archive and the
existing updater replaces the whole package, including the `ui/` directory.
Node.js is a build-time dependency only. Do not remove the native adapters
until physical Windows and macOS acceptance verifies tray reopening, autostart,
Mac ↔ Windows sync, pairing, history, updates, high DPI and accessibility.

## Validation limits

Playwright covers deterministic browser states, not WebView2/WKWebView system
integration. Current macOS smoke uses a temporary config and does not perform
real pairing or clipboard sync. A Windows executable can be cross-built on
macOS, but that does not validate its tray, window activation, WebView2 runtime
or scaling. Wails v3 remains a pinned beta and increases runtime footprint;
measure actual RAM/CPU on both OSes before replacing the native GUI in a
release. No release is produced by this branch.
