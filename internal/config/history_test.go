package config

import (
	"strings"
	"testing"
	"time"
)

func TestHistoryDefaults(t *testing.T) {
	c, e := Parse(strings.NewReader(valid))
	if e != nil {
		t.Fatal(e)
	}
	if c.History.Enabled {
		t.Fatal("history must be disabled by default")
	}
	if c.HistoryRetention() != HistoryDefaultRetention || c.HistoryMaxItems() != HistoryDefaultItems {
		t.Fatal("unexpected history defaults", c.History)
	}
	if c.History.Retention != HistoryDefaultRetentionS || c.History.MaxItems != HistoryDefaultItems {
		t.Fatal("history defaults not applied", c.History)
	}
}

func TestHistoryRetentionBounds(t *testing.T) {
	cases := map[string]bool{"29m": false, "30m": true, "1h": true, "24h": true, "72h": true, "72h1s": false, "nope": false, "": true}
	for value, ok := range cases {
		c, e := Parse(strings.NewReader(valid + "history:\n  enabled: true\n  retention: " + value + "\n"))
		if ok != (e == nil) {
			t.Fatalf("retention %q: err=%v", value, e)
		}
		if ok && value != "" && c.HistoryRetention().String() != mustDuration(t, value) {
			t.Fatalf("retention %q normalized to %v", value, c.HistoryRetention())
		}
	}
	// Zero value falls back to the default instead of failing validation.
	var empty Config
	empty.Security.Secret = strings.Repeat("x", 32)
	empty.Device.ID = "mac"
	empty.Listen.Address = "127.0.0.1"
	if empty.HistoryRetention() != HistoryDefaultRetention || empty.HistoryMaxItems() != HistoryDefaultItems {
		t.Fatal("zero history values must fall back to defaults")
	}
}

func TestHistoryMaxItemsBounds(t *testing.T) {
	cases := map[int]bool{2: false, 3: true, 10: true, 50: true, 51: false, -1: false}
	for count, ok := range cases {
		_, e := Parse(strings.NewReader(valid + "history:\n  enabled: true\n  max_items: " + itoa(count) + "\n"))
		if ok != (e == nil) {
			t.Fatalf("max_items %d: err=%v", count, e)
		}
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	var c Config
	c.Security.Secret = strings.Repeat("x", 32)
	c.Device.ID = "mac"
	c.Listen.Address = "127.0.0.1"
	c.Listen.Port = 45873
	c.Sync.Mode = "bidirectional"
	c.History = History{Enabled: true, Retention: "6h", MaxItems: 25}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	if c.HistoryRetention() != 6*time.Hour || c.HistoryMaxItems() != 25 {
		t.Fatal("history settings lost")
	}
}

func mustDuration(t *testing.T, s string) string {
	t.Helper()
	switch s {
	case "30m":
		return "30m0s"
	case "1h":
		return "1h0m0s"
	case "6h":
		return "6h0m0s"
	case "12h":
		return "12h0m0s"
	case "24h":
		return "24h0m0s"
	case "48h":
		return "48h0m0s"
	case "72h":
		return "72h0m0s"
	}
	t.Fatalf("unexpected duration %q", s)
	return ""
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
