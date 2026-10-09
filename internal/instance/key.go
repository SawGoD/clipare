// Package instance coordinates local GUI activation, independently of NetBird.
package instance

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
)

func Key(path string) string {
	full, err := filepath.Abs(path)
	if err != nil {
		full = filepath.Clean(path)
	}
	return fmt.Sprintf("Clipare.GUI.%x", sha256.Sum256([]byte(strings.ToLower(full))))
}
