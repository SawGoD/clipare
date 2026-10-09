package instance

import (
	"path/filepath"
	"testing"
)

func TestProfileKey(t *testing.T) {
	if Key("./fixture.yaml") != Key("fixture.yaml") {
		t.Fatal("equivalent paths differ")
	}
	if Key(filepath.Join(t.TempDir(), "a.yaml")) == Key(filepath.Join(t.TempDir(), "b.yaml")) {
		t.Fatal("different profiles collide")
	}
}
