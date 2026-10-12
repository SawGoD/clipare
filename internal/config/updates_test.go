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

func TestUpdateChannelPersistenceAndValidation(t *testing.T) {
	c, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if c.Updates.Channel != "" {
		t.Fatal("old installs must infer channel")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	for _, channel := range []string{"stable", "beta"} {
		c.Updates.Channel = channel
		if err := Save(path, c); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil || got.Updates.Channel != channel || got.Identity != c.Identity {
			t.Fatal("channel persistence", err)
		}
	}
	c.Updates.Channel = "nightly"
	if err := c.Validate(); err == nil {
		t.Fatal("invalid channel")
	}
}
