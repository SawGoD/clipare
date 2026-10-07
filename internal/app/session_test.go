package app

import (
	"clipare/internal/config"
	"clipare/internal/security"
	"clipare/internal/transport"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strconv"
	"syscall"
	"testing"
	"time"
)

type fakeClipboard struct{ failRead, failWatch bool }

func (f *fakeClipboard) Read() (string, bool, error) {
	if f.failRead {
		return "", false, errors.New("failed")
	}
	return "", true, nil
}

func TestListenErrorClassification(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: syscall.EADDRNOTAVAIL}
	if !errors.Is(listenError(wrapped), ErrAddressUnavailable) {
		t.Fatal("missing address not recoverable")
	}
	if errors.Is(listenError(syscall.EADDRINUSE), ErrAddressUnavailable) {
		t.Fatal("occupied port must not silently retry")
	}
}
func (*fakeClipboard) Write(string) error { return nil }
func (f *fakeClipboard) Watch(ctx context.Context, ch chan<- struct{}) error {
	if f.failWatch {
		return errors.New("watcher failed")
	}
	<-ctx.Done()
	return nil
}
func TestSessionLifecycleAndRestart(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	_, port, _ := net.SplitHostPort(l.Addr().String())
	l.Close()
	c, _ := config.Default()
	c.Listen.Address = "127.0.0.1"
	c.Listen.Port, _ = strconv.Atoi(port)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := transport.NewClient(security.StaticSecret(c.Security.Secret))
	defer client.Close()
	for i := 0; i < 3; i++ {
		s, e := Start(context.Background(), c, &fakeClipboard{}, log, nil)
		if e != nil {
			t.Fatal(e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		h, e := client.Health(ctx, "http://"+c.ListenAddress())
		cancel()
		if e != nil || h.Device != c.Device.ID {
			t.Fatal(e)
		}
		if e = s.Stop(); e != nil {
			t.Fatal(e)
		}
		select {
		case <-s.Done():
		default:
			t.Fatal("session leaked")
		}
	}
	if _, e = Start(context.Background(), c, &fakeClipboard{failRead: true}, log, nil); e == nil {
		t.Fatal("initialization failure ignored")
	}
	s, e := Start(context.Background(), c, &fakeClipboard{failWatch: true}, log, nil)
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-s.Done():
		if s.Err() == nil {
			t.Fatal("watcher failure ignored")
		}
	case <-time.After(time.Second):
		t.Fatal("watcher failure left server running")
	}
}
