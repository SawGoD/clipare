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
