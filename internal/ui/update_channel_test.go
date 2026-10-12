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

type channelTestDesktop struct {
	*testDesktop
	cache, channel string
	phase          int
}

func (*channelTestDesktop) UpdateSettings(string, bool)       {}
func (*channelTestDesktop) UpdateEnabled() bool               { return false }
func (*channelTestDesktop) UpdatePrompt(updatePrompt)         {}
func (*channelTestDesktop) UpdateClose()                      {}
func (d *channelTestDesktop) UpdateCacheDir() (string, error) { return d.cache, nil }
func (*channelTestDesktop) UpdateChannel() string             { return "beta" }
func (d *channelTestDesktop) UpdateChannelSettings(channel string) {
	d.channel = channel
	if d.phase == 1 && channel == "beta" {
		d.phase = 2
		d.actions <- eventQuit
	}
}
func (d *channelTestDesktop) Update(_ string, enabled bool, _ []config.Peer, _ map[string]bool) {
	if enabled && d.phase == 0 {
		d.phase = 1
		d.f.Values[3] = "invalid draft port"
		d.actions <- eventUpdateChannel
	}
}

func TestChannelPreferencePersistsWithoutSyncRestartOrDraftSave(t *testing.T) {
	c, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	c.Listen.Address = "127.0.0.1"
	c.Updates.Enabled = false
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.Save(path, c); err != nil {
		t.Fatal(err)
	}
	d := &channelTestDesktop{testDesktop: &testDesktop{f: toForm(c), actions: make(chan int, 4), path: path}, cache: t.TempDir()}
	starts := 0
	start := func(ctx context.Context, c config.Config, b clipboard.Backend, log *slog.Logger, h func(string, bool)) (*app.Session, error) {
		starts++
		c.Listen.Port = 0
		return app.Start(ctx, c, b, log, h)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runDesktop(ctx, path, slog.New(slog.NewTextHandler(io.Discard, nil)), d, testClipboard{}, start, loopTiming{time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(path)
	if err != nil || got.Updates.Channel != "beta" || got.Listen != c.Listen || got.Identity != c.Identity || starts != 1 || d.phase != 2 || len(d.alerts) != 0 {
		t.Fatal("channel changed unrelated state", err, starts, d.phase, d.alerts)
	}
}
