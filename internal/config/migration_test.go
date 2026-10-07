package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigration(t *testing.T) {
	c, err := Parse(strings.NewReader(valid))
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err = Save(p, c); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(p)
	n, err := LoadMigrated(p)
	if err != nil {
		t.Fatal(err)
	}
	if n.SchemaVersion != 2 || n.Device.ID != c.Device.ID || !n.Peers[0].Legacy || n.Peers[0].LegacySecret != c.Security.Secret {
		t.Fatal("legacy configuration lost")
	}
	backup, _ := os.ReadFile(p + ".v1.backup")
	if !bytes.Equal(original, backup) {
		t.Fatal("backup changed")
	}
	again, err := LoadMigrated(p)
	if err != nil || again.Identity.PrivateKey != n.Identity.PrivateKey {
		t.Fatal("identity regenerated")
	}
	if err = Save(p, c); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadMigrated(p); err == nil {
		t.Fatal("overwrote backup")
	}
	got, _ := os.ReadFile(p)
	if !bytes.Equal(original, got) {
		t.Fatal("failed migration changed config")
	}
}
