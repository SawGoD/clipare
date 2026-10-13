//go:build windows

package ui

import "fmt"

func reactSystemAccent() string {
	v := systemTheme().accent
	return fmt.Sprintf("#%02x%02x%02x", byte(v), byte(v>>8), byte(v>>16))
}
