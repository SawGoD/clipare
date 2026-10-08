//go:build !windows

package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnwritableInstallation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	old, new := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(old, "app"), []byte("old"), 0600)
	os.WriteFile(filepath.Join(new, "app"), []byte("new"), 0600)
	os.Chmod(old, 0500)
	defer os.Chmod(old, 0700)
	if replaceTransaction(new, old, []string{"app"}, os.Rename, func() error { return nil }) == nil {
		t.Fatal("permissions ignored")
	}
	b, _ := os.ReadFile(filepath.Join(old, "app"))
	if string(b) != "old" {
		t.Fatal("old changed")
	}
}
