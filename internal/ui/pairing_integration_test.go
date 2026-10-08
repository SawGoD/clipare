package ui

import (
	"clipare/internal/app"
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"
)

type testPairDesktop struct{ *testDesktop }

func (*testPairDesktop) Discovered(string, string) {}
func (*testPairDesktop) DiscoveredSelected() int   { return -1 }
func (*testPairDesktop) Pair(string, string, bool) {}
func (*testPairDesktop) PairClose()                {}
func TestPairingDesktopJoinsShutdown(t *testing.T) {
	c, _ := config.Default()
	c.Listen.Address = "127.0.0.1"
	c.Listen.Port = 45873
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.Save(path, c); err != nil {
		t.Fatal(err)
	}
	d := &testPairDesktop{&testDesktop{actions: make(chan int, 4), path: path}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Use a synthetic session starter, no NetBird CLI, clipboard or login items.
	start := func(ctx context.Context, c config.Config, b clipboard.Backend, log *slog.Logger, h func(string, bool)) (*app.Session, error) {
		c.Listen.Port = 0
		return app.Start(ctx, c, b, log, h)
	}
	if err := runDesktop(ctx, path, slog.New(slog.NewTextHandler(io.Discard, nil)), d, testClipboard{}, start, loopTiming{time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if !d.running || len(d.alerts) > 0 {
		t.Fatal("pairing GUI failed startup")
	}
}
