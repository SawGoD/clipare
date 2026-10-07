package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAndPair(t *testing.T) {
	a, _ := Default()
	b, _ := Default()
	a.Listen.Address = "100.64.1.1"
	b.Listen.Address = "100.64.1.2"
	paired, e := AddConnection(b, ConnectionCode(a))
	if e != nil || paired.Security.Secret != a.Security.Secret || len(paired.Peers) != 1 {
		t.Fatal(e)
	}
	back, e := AddConnection(a, ConnectionCode(paired))
	if e != nil || back.Peers[0].ID != b.Device.ID {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if e = Save(path, back); e != nil {
		t.Fatal(e)
	}
	if e = Save(path, paired); e != nil {
		t.Fatal(e)
	}
	got, e := Load(path)
	if e != nil || got.Device.ID != b.Device.ID {
		t.Fatal(e)
	}
	if Platform() != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Fatal("permissions")
		}
	}
	c, _ := Default()
	if _, e = AddConnection(paired, ConnectionCode(c)); e == nil {
		t.Fatal("group secret overwritten")
	}
	if _, e = AddConnection(a, ConnectionCode(a)); e == nil {
		t.Fatal("self pair")
	}
}
