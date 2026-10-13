//go:build !windows && (!darwin || !cgo)

package ui

import (
	"context"
	"errors"
	"log/slog"
)

func RunReactWithReady(context.Context, string, *slog.Logger, func() error, bool) error {
	return errors.New("React desktop UI is supported only on macOS and Windows")
}
func HasPackagedReactUI() bool { return false }
