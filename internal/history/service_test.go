package history

import (
	"clipare/internal/config"
	"context"
	"io"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type memoryKeystore struct {
	mu   sync.Mutex
	key  []byte
	fail bool
}

func (m *memoryKeystore) Load() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return nil, errors.New("keystore broken")
	}
	if m.key == nil {
		return nil, os.ErrNotExist
	}
	return m.key, nil
}

func (m *memoryKeystore) Save(key []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("keystore broken")
	}
	m.key = append([]byte(nil), key...)
	return nil
}

func serviceConfig(enabled bool) config.Config {
	var c config.Config
	c.Device.ID = "mac"
	c.Device.Name = "MacBook"
	c.Listen.Address = "127.0.0.1"
	c.Listen.Port = 45873
	c.Security.Secret = "abcdefghijklmnopqrstuvwxyz012345"
	c.Sync.Mode = "bidirectional"
	c.History = config.History{Enabled: enabled, Retention: "24h", MaxItems: 10}
	c.Peers = []config.Peer{{ID: "desktop", Name: "Desktop", Address: "desktop.netbird.cloud", Port: 45873}}
	return c
}

func openTestService(t *testing.T, c config.Config) (*Service, *memoryKeystore, string) {
	t.Helper()
	dir := t.TempDir()
	ks := &memoryKeystore{}
	s, e := OpenWithKeystore(filepath.Join(dir, HistoryFile), ks, c, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s, ks, dir
}

func TestServiceGeneratesAndKeepsKey(t *testing.T) {
	dir := t.TempDir()
	ks := &memoryKeystore{}
	path := filepath.Join(dir, HistoryFile)
	s, e := OpenWithKeystore(path, ks, serviceConfig(true), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	if ks.key == nil || len(ks.key) != keySize {
		t.Fatal("master key not generated")
	}
	s.Record("id-1", "mac", "hello", 1700000000, true)
	s.Close()
	// Reopening with the same keystore must keep history readable.
	reopened, e := OpenWithKeystore(path, ks, serviceConfig(true), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	items, e := reopened.Items()
	if e != nil || len(items) != 1 || items[0].Text != "hello" {
		t.Fatal("history unreadable after restart", items, e)
	}
}

func TestServiceRecordsLocalAndRemote(t *testing.T) {
	s, _, _ := openTestService(t, serviceConfig(true))
	s.Record("id-1", "mac", "local text", 1700000000, true)
	s.Record("id-2", "desktop", "remote text", 1700000001, false)
	items, e := s.Items()
	if e != nil || len(items) != 2 {
		t.Fatal(items, e)
	}
	if items[1].OriginName != "MacBook" || items[1].OriginID != "mac" {
		t.Fatal("local origin wrong", items[1])
	}
	if items[0].OriginName != "Desktop" || items[0].OriginID != "desktop" {
		t.Fatal("remote origin wrong", items[0])
	}
}

func TestServiceDisabledRecordsNothing(t *testing.T) {
	s, _, _ := openTestService(t, serviceConfig(false))
	s.Record("id-1", "mac", "local text", 1700000000, true)
	s.Record("id-2", "desktop", "remote text", 1700000001, false)
	items, e := s.Items()
	if e != nil || len(items) != 0 {
		t.Fatal("history recorded while disabled", items, e)
	}
}

func TestServiceUnknownPeerFallsBackToID(t *testing.T) {
	s, _, _ := openTestService(t, serviceConfig(true))
	s.Record("id-1", "laptop", "text", 1700000000, false)
	items, _ := s.Items()
	if items[0].OriginName != "laptop" {
		t.Fatal("unexpected origin name", items[0])
	}
}

func TestServiceSensitiveAndURLClassification(t *testing.T) {
	s, _, _ := openTestService(t, serviceConfig(true))
	s.Record("id-1", "mac", "ghp_0123456789abcdefghijklmnopqrstuvwxyzAB", 1700000000, true)
	s.Record("id-2", "mac", "https://github.com/SawGoD/clipare", 1700000001, true)
	s.Record("id-3", "mac", "просто заметка", 1700000002, true)
	items, _ := s.Items()
	if !items[2].Sensitive || items[2].Type != ContentText {
		t.Fatal("secret not flagged", items[2])
	}
	if items[1].Type != ContentURL || items[1].Sensitive {
		t.Fatal("URL misclassified", items[1])
	}
	if items[0].Sensitive {
		t.Fatal("ordinary text flagged", items[0])
	}
}

func TestServiceKeyLostThenClear(t *testing.T) {
	dir := t.TempDir()
	ks := &memoryKeystore{}
	path := filepath.Join(dir, HistoryFile)
	s, e := OpenWithKeystore(path, ks, serviceConfig(true), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	s.Record("id-1", "mac", "hello", 1700000000, true)
	s.Close()
	// The OS keystore now returns a different key (profile reset).
	replaced := &memoryKeystore{key: []byte("ffffffffffffffffffffffffffffffff")}
	reopened, e := OpenWithKeystore(path, replaced, serviceConfig(true), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if _, e = reopened.Items(); !errors.Is(e, ErrKeyLost) {
		t.Fatal("expected key-lost error", e)
	}
	// The app must not crash and must be able to offer clearing.
	if e = reopened.Clear(); e != nil {
		t.Fatal(e)
	}
	items, e := reopened.Items()
	if e != nil || len(items) != 0 {
		t.Fatal("clear after key loss failed", items, e)
	}
	reopened.Record("id-2", "mac", "fresh", 1700000001, true)
	if items, e = reopened.Items(); e != nil || len(items) != 1 || items[0].Text != "fresh" {
		t.Fatal("recording after reset failed", items, e)
	}
}

func TestServiceRunPeriodicCleanup(t *testing.T) {
	s, _, _ := openTestService(t, serviceConfig(true))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	cancel()
	<-done
	// Run must terminate promptly and not panic.
}

func TestDefaultPath(t *testing.T) {
	if got := DefaultPath("/home/user/.config/Clipare/config.yaml"); got != "/home/user/.config/Clipare/history.db" {
		t.Fatal(got)
	}
}
