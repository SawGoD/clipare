# Desktop UI design baseline

Clipare uses one state model but two platform renderers. Do not force identical
controls, spacing or window materials across Windows and macOS. Keep the main
workflow compact: status, devices/history, additional settings, version. Advanced
controls appear only when the header gear is expanded.

## Windows 11

Use Fluent UI React v9 controls and its light/dark theme tokens. Build hierarchy
from the [Fluent 2 spacing ramp](https://fluent2.microsoft.design/layout) rather
than aligning individual pixels; retain system accent and contrast behavior.
Keep motion short and tied to a user action, following [Windows motion guidance](https://learn.microsoft.com/en-us/windows/apps/design/signature-experiences/motion).
Fluent's [Dropdown guidance](https://fluent2.microsoft.design/components/web/react/core/dropdown/usage)
recommends `inlinePopup` when possible; use it inside Clipare's inline settings
instead of a body-level portal. A browser preview is not a WebView2 acceptance
test: retest choices, scrolling and focus on Windows 11 before a release.

## macOS

Use Apple system typography, appearance and accent. [Apple's materials guidance](https://developer.apple.com/design/human-interface-guidelines/materials)
reserves Liquid Glass primarily for controls/navigation; content should remain
legible and use standard materials. The native Wails host provides an
`NSGlassEffectView` backdrop on macOS 26 and `NSVisualEffectView` fallback on
older systems. This requires Wails' `private_mac_apis` build tag to make WKWebView
transparent. That tag relies on an undocumented WebKit property and must be
revalidated on macOS updates and before notarized distribution. Respect Reduce
Transparency with opaque content surfaces. Browser preview shows a solid fallback,
not a simulation of native glass.

## Validation

Use `cd frontend && npm run test:e2e` for deterministic states/screenshots.
Inspect screenshots after each visual change. Before publishing, manually verify
Windows WebView2 and a packaged macOS app in light/dark, accent and accessibility
settings; test pairing, history, updates and tray lifecycle on actual devices.
