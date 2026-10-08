//go:build !windows

package update

import "os"

func replaceState(from, to string) error { return os.Rename(from, to) }
