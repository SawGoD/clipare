package config

import (
	"strings"
	"testing"
)

const valid = "device:\n  id: mac\nlisten:\n  address: 127.0.0.1\nsecurity:\n  secret: abcdefghijklmnopqrstuvwxyz012345\npeers:\n  - id: pc\n    address: pc.netbird.cloud\n    port: 45873\n"

func TestParse(t *testing.T) {
	c, e := Parse(strings.NewReader(valid))
	if e != nil || c.Mode() != "bidirectional" || c.Peers[0].URL() != "http://pc.netbird.cloud:45873" {
		t.Fatal(c, e)
	}
	for _, s := range []string{valid + "unknown: secret\n", strings.Replace(valid, "127.0.0.1", "", 1), strings.Replace(valid, "abcdefghijklmnopqrstuvwxyz012345", "CHANGE_ME", 1), valid + "---\nx: y\n", strings.Replace(valid, "id: pc", "id: mac", 1)} {
		if _, e := Parse(strings.NewReader(s)); e == nil {
			t.Fatal("accepted invalid config")
		}
	}
}
