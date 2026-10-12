package sync

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type fakeBackend struct {
	mu      sync.Mutex
	value   string
	present bool
}

func (b *fakeBackend) Read() (string, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.value, b.present, nil
}
func (b *fakeBackend) Write(s string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.value = s
	b.present = true
	return nil
}
func (b *fakeBackend) Watch(ctx context.Context, ch chan<- struct{}) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestManagerObserverSeesLocalAndRemote(t *testing.T) {
	b := &fakeBackend{}
	var messages []Message
	done := make(chan struct{})
	m := NewManager(b, "mac", "bidirectional", slog.New(slog.NewTextHandler(discard{}, nil)), func(msg Message) {
		messages = append(messages, msg)
	})
	var events []struct {
		id    string
		local bool
	}
	m.SetObserver(func(msg Message, local bool) {
		events = append(events, struct {
			id    string
			local bool
		}{msg.ID, local})
		if len(events) == 2 {
			close(done)
		}
	})
	if e := m.Initialize(); e != nil {
		t.Fatal(e)
	}
	b.Write("local text")
	m.LocalChanged()
	e := m.Receive(context.Background(), Message{Version: 1, ID: mustUUID(t), Source: "desktop", Timestamp: time.Now().Add(2 * time.Second).Unix(), MIME: "text/plain", Data: "remote text"})
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("observer did not see both events")
	}
	if !events[0].local || events[1].local {
		t.Fatal("unexpected local flags", events)
	}
}

func TestManagerObserverSkipsDuplicatesAndCodes(t *testing.T) {
	b := &fakeBackend{}
	m := NewManager(b, "mac", "bidirectional", slog.New(slog.NewTextHandler(discard{}, nil)), func(Message) {})
	calls := 0
	m.SetObserver(func(Message, bool) { calls++ })
	if e := m.Initialize(); e != nil {
		t.Fatal(e)
	}
	// Connection codes must never reach the observer.
	b.Write("clipare1:abc")
	m.LocalChanged()
	if calls != 0 {
		t.Fatal("connection code observed")
	}
	// Repeated receive of the same message is deduplicated before observing.
	msg := Message{Version: 1, ID: mustUUID(t), Source: "desktop", Timestamp: time.Now().Unix(), MIME: "text/plain", Data: "same"}
	if e := m.Receive(context.Background(), msg); e != nil {
		t.Fatal(e)
	}
	if e := m.Receive(context.Background(), msg); e != nil {
		t.Fatal(e)
	}
	if calls != 1 {
		t.Fatal("duplicate message observed", calls)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func mustUUID(t *testing.T) string {
	t.Helper()
	msg, e := NewMessage("mac", "probe")
	if e != nil {
		t.Fatal(e)
	}
	return msg.ID
}
