package main

import (
	"archive/zip"
	"clipare"
	"os"
	"path/filepath"
	"testing"
)

func TestVersion(t *testing.T) {
	if !semver.MatchString(clipare.Version()) {
		t.Fatal("VERSION must be stable SemVer")
	}
	for _, s := range []string{"01.2.3", "v1.2.3", "1.2", "1.2.3; command"} {
		if semver.MatchString(s) {
			t.Fatal(s)
		}
	}
}
func TestArchivePreservesBundleAndMode(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	binary := filepath.Join(stage, "Clipare.app", "Contents", "MacOS", "Clipare")
	if e := os.MkdirAll(filepath.Dir(binary), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(binary, []byte("binary"), 0755); e != nil {
		t.Fatal(e)
	}
	archive := filepath.Join(root, "app.zip")
	if e := archiveDir(stage, archive); e != nil {
		t.Fatal(e)
	}
	r, e := zip.OpenReader(archive)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if len(r.File) != 1 || r.File[0].Name != "Clipare.app/Contents/MacOS/Clipare" {
		t.Fatal("bundle structure")
	}
	if r.File[0].Mode().Perm()&0111 == 0 && os.PathSeparator != '\\' {
		t.Fatal("executable bit lost")
	}
}
