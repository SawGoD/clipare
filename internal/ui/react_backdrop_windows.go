//go:build windows

package ui

import "github.com/wailsapp/wails/v3/pkg/w32"

// Only enable a system backdrop where the documented DWM backdrop API exists.
func reactWindowsBackdropSupported() bool { return w32.SupportsBackdropTypes() }
