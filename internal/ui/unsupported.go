//go:build (!windows && !darwin) || (darwin && !cgo)

package ui

import "errors"

func newDesktop() (desktop, error) { return nil, errors.New("GUI requires Windows or macOS with CGO") }
