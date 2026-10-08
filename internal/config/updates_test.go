package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateDefaultsAndPersistence(t *testing.T) {
	c, e := Default()
	if e != nil || !c.Updates.Enabled {
		t.Fatal("default", e)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if e = Save(path, c); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(path)
	old := strings.Replace(string(b), "updates:\n    enabled: true\n", "", 1)
	parsed, e := Parse(strings.NewReader(old))
	if e != nil || !parsed.Updates.Enabled {
		t.Fatal("old config default", e)
	}
	c.Updates.Enabled = false
	Save(path, c)
	got, e := Load(path)
	if e != nil || got.Updates.Enabled {
		t.Fatal("disabled persistence", e)
	}
	if got.Identity != c.Identity || got.Device != c.Device || got.Group != c.Group {
		t.Fatal("identity changed")
	}
}
