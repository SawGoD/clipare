//go:build !windows && !darwin

package autostart

import "errors"

func Set(bool, string, string) error { return errors.New("autostart unsupported on this OS") }
