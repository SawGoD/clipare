package ui

import (
	"clipare/internal/app"
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"context"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type testClipboard struct{}

func (testClipboard) Read() (string, bool, error)                         { return "", true, nil }
func (testClipboard) Write(string) error                                  { return nil }
func (testClipboard) Watch(ctx context.Context, ch chan<- struct{}) error { <-ctx.Done(); return nil }

type testDesktop struct {
	f                 form
	actions           chan int
	waiting, running  bool
	alerts            []string
	path              string
	initial           config.Config
	savedWhileWaiting bool
	expectedID        string
}

func (*testDesktop) Init() error { return nil }
func (d *testDesktop) Poll() int {
	select {
	case e := <-d.actions:
		return e
	default:
		return eventNone
	}
}
func (d *testDesktop) Show(f form, _ []config.Peer, _ []string) {
	d.f = f
	if d.initial.Device.ID != "" {
		d.expectedID = f.Values[1]
		d.f = toForm(d.initial)
		d.actions <- eventSave
		d.initial = config.Config{}
	}
}
func (d *testDesktop) Read() form        { return d.f }
func (*testDesktop) Selected() int       { return -1 }
func (*testDesktop) SetPeer(config.Peer) {}
func (d *testDesktop) Update(status string, enabled bool, _ []config.Peer, _ map[string]bool) {
	if status == "Ожидание NetBird: локальный IP ещё недоступен" {
		d.waiting = true
		_, e := config.Load(d.path)
		d.savedWhileWaiting = e == nil
	}
	if enabled && !d.running {
		d.running = true
		d.actions <- eventQuit
	}
}
func (d *testDesktop) Alert(s string) { d.alerts = append(d.alerts, s) }
func (*testDesktop) Close()           {}

type preparedTestDesktop struct {
	testDesktop
	prepared bool
}

func (d *preparedTestDesktop) Prepare(f form, _ []config.Peer, _ []string) {
	d.f, d.prepared = f, true
}

func TestTrayStartupPreparesPersistedFieldsBeforeReady(t *testing.T) {
	c, _ := config.Default()
	c.Sync.Enabled = false
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.Save(path, c); err != nil {
		t.Fatal(err)
	}
	d := &preparedTestDesktop{testDesktop: testDesktop{path: path, actions: make(chan int, 1)}}
	d.actions <- eventQuit
	ready := func() error {
		if !d.prepared || d.Read().Values != toForm(c).Values {
			t.Fatal("startup did not prepare saved fields")
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runDesktopReady(ctx, path, slog.New(slog.NewTextHandler(io.Discard, nil)), d, testClipboard{}, nil, loopTiming{time.Millisecond, time.Millisecond, time.Millisecond}, ready, false); err != nil {
		t.Fatal(err)
	}
}

func TestGUIRecoversAndPersistsWithoutNetBird(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	listener.Close()
	c, _ := config.Default()
	c.Listen.Address = "127.0.0.1"
	c.Listen.Port, _ = strconv.Atoi(port)
	path := filepath.Join(t.TempDir(), "config.yaml")
	d := &testDesktop{initial: c, actions: make(chan int, 4), path: path}
	calls := 0
	start := func(ctx context.Context, c config.Config, b clipboard.Backend, log *slog.Logger, h func(string, bool)) (*app.Session, error) {
		calls++
		if calls == 1 {
			return nil, app.ErrAddressUnavailable
		}
		return app.Start(ctx, c, b, log, h)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if e = runDesktop(ctx, path, slog.New(slog.NewTextHandler(io.Discard, nil)), d, testClipboard{}, start, loopTiming{time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond}); e != nil {
		t.Fatal(e)
	}
	if !d.waiting || !d.running || !d.savedWhileWaiting || calls != 2 || len(d.alerts) != 0 {
		t.Fatalf("recovery failed: waiting=%v running=%v saved=%v calls=%d alerts=%v", d.waiting, d.running, d.savedWhileWaiting, calls, d.alerts)
	}
	saved, e := config.Load(path)
	if e != nil || saved.Device.ID != d.expectedID {
		t.Fatal("settings were lost", e)
	}
}
