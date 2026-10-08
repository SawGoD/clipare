package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReplacementAndRollback(t *testing.T) {
	for _, fail := range []string{"none", "replacement", "startup"} {
		t.Run(fail, func(t *testing.T) {
			old, new := t.TempDir(), t.TempDir()
			for _, p := range []string{"Clipare.exe", "clipare-console.exe"} {
				os.WriteFile(filepath.Join(old, p), []byte("old"), 0600)
				os.WriteFile(filepath.Join(new, p), []byte("new"), 0600)
			}
			os.WriteFile(filepath.Join(old, "config.yaml"), []byte("do not change"), 0600)
			calls := 0
			rename := func(a, b string) error {
				calls++
				if fail == "replacement" && calls == 4 {
					return errors.New("injected rename failure")
				}
				return os.Rename(a, b)
			}
			e := replaceTransaction(new, old, []string{"Clipare.exe", "clipare-console.exe"}, rename, func() error {
				if fail == "startup" {
					return errors.New("injected startup failure")
				}
				return nil
			})
			want := "new"
			if fail != "none" {
				want = "old"
				if e == nil {
					t.Fatal("missing failure")
				}
			} else if e != nil {
				t.Fatal(e)
			}
			for _, p := range []string{"Clipare.exe", "clipare-console.exe"} {
				b, _ := os.ReadFile(filepath.Join(old, p))
				if string(b) != want {
					t.Fatalf("%s: %s", p, b)
				}
			}
			b, _ := os.ReadFile(filepath.Join(old, "config.yaml"))
			if string(b) != "do not change" {
				t.Fatal("config modified")
			}
			entries, _ := os.ReadDir(old)
			for _, entry := range entries {
				if entry.IsDir() {
					t.Fatal("transaction residue")
				}
			}
		})
	}
}
func TestPreparationFailureLeavesOld(t *testing.T) {
	old, new := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(old, "Clipare.exe"), []byte("old"), 0600)
	if replaceTransaction(new, old, []string{"Clipare.exe"}, os.Rename, func() error { return nil }) == nil {
		t.Fatal("missing new exe")
	}
	b, _ := os.ReadFile(filepath.Join(old, "Clipare.exe"))
	if string(b) != "old" {
		t.Fatal("old lost")
	}
}
func TestRollbackFailureRetainsBackup(t *testing.T) {
	old, new := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(old, "app"), []byte("old"), 0600)
	os.WriteFile(filepath.Join(new, "app"), []byte("new"), 0600)
	n := 0
	rename := func(a, b string) error {
		n++
		if n >= 4 {
			return errors.New("disk failed")
		}
		return os.Rename(a, b)
	}
	if replaceTransaction(new, old, []string{"app"}, rename, func() error { return errors.New("startup") }) == nil {
		t.Fatal("failure")
	}
	entries, _ := os.ReadDir(old)
	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			b, _ := os.ReadFile(filepath.Join(old, entry.Name(), "backup", "app"))
			found = string(b) == "old"
		}
	}
	if !found {
		t.Fatal("backup lost")
	}
}
func TestCleanup(t *testing.T) {
	cache := t.TempDir()
	now := time.Now()
	for _, name := range []string{"stage-stale", "stage-recent", "stage-unmarked", "unrelated"} {
		os.Mkdir(filepath.Join(cache, name), 0700)
	}
	for _, name := range []string{"stage-stale", "stage-recent"} {
		created := now
		if name == "stage-stale" {
			created = now.Add(-8 * 24 * time.Hour)
		}
		b, _ := json.Marshal(struct {
			Created time.Time `json:"created"`
		}{created})
		os.WriteFile(filepath.Join(cache, name, ".clipare-stage"), b, 0600)
	}
	Cleanup(cache, now)
	if _, e := os.Stat(filepath.Join(cache, "stage-stale")); !os.IsNotExist(e) {
		t.Fatal("stale not removed")
	}
	for _, name := range []string{"stage-recent", "stage-unmarked", "unrelated"} {
		if _, e := os.Stat(filepath.Join(cache, name)); e != nil {
			t.Fatal(name)
		}
	}
}
