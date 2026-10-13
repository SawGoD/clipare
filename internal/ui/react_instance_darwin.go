//go:build darwin && cgo

package ui

import "github.com/wailsapp/wails/v3/pkg/application"

func installReactActivation(*application.WebviewWindow, string) (func(), error) {
	return func() {}, nil
}
