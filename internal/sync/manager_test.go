package sync

import (
	"clipare/internal/clipboard"
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
)

type fake struct {
	mu     sync.Mutex
	text   string
	writes int
	fail   bool
}

func (f *fake) Read() (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.text, true, nil
}
func (f *fake) Write(s string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("failed")
	}
	f.text = s
	f.writes++
	return nil
}
func (f *fake) Watch(ctx context.Context, ch chan<- struct{}) error { <-ctx.Done(); return nil }

var _ clipboard.Backend = (*fake)(nil)

func TestThreePeerLoopAndEcho(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	var sent []Message
	fa, fb, fc := &fake{}, &fake{}, &fake{}
	a := NewManager(fa, "a", "bidirectional", log, func(m Message) { sent = append(sent, m) })
	b := NewManager(fb, "b", "bidirectional", log, func(Message) { t.Error("remote echo") })
	c := NewManager(fc, "c", "bidirectional", log, func(Message) { t.Error("remote echo") })
	for _, m := range []*Manager{a, b, c} {
		if e := m.Initialize(); e != nil {
			t.Fatal(e)
		}
	}
	fa.text = "hello"
	a.LocalChanged()
	if len(sent) != 1 {
		t.Fatal(sent)
	}
	x := sent[0]
	for _, m := range []*Manager{b, c, a, b, c} {
		if e := m.Receive(context.Background(), x); e != nil {
			t.Fatal(e)
		}
		m.LocalChanged()
	}
	if fa.writes != 0 || fb.writes != 1 || fc.writes != 1 {
		t.Fatal("duplicate writes")
	}
	if len(sent) != 1 {
		t.Fatal("loop")
	}
}
func TestConcurrentReceiveAndFailedWrite(t *testing.T) {
	f := &fake{fail: true}
	m := NewManager(f, "b", "bidirectional", slog.New(slog.NewTextHandler(io.Discard, nil)), func(Message) {})
	x, _ := NewMessage("a", "text")
	if m.Receive(context.Background(), x) == nil {
		t.Fatal("failure expected")
	}
	f.fail = false
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := m.Receive(context.Background(), x); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if f.writes != 1 {
		t.Fatal(f.writes)
	}
	disabled := NewManager(f, "c", "disabled", m.log, func(Message) {})
	if !errors.Is(disabled.Receive(context.Background(), x), ErrDisabled) {
		t.Fatal("disabled")
	}
}
func TestSimultaneousChangesConverge(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	fa, fb := &fake{}, &fake{}
	var xa, xb Message
	a := NewManager(fa, "a", "bidirectional", log, func(m Message) { xa = m })
	b := NewManager(fb, "b", "bidirectional", log, func(m Message) { xb = m })
	a.Initialize()
	b.Initialize()
	fa.text = "one"
	fb.text = "two"
	a.LocalChanged()
	b.LocalChanged()
	a.Receive(context.Background(), xb)
	b.Receive(context.Background(), xa)
	if fa.text != fb.text {
		t.Fatal("divergence")
	}
}

func TestRemoteWriteWithPendingLocalEvent(t *testing.T) {
	f := &fake{text: "old"}
	m := NewManager(f, "b", "bidirectional", slog.New(slog.NewTextHandler(io.Discard, nil)), func(Message) { t.Error("remote echo") })
	m.Initialize()
	f.text = "unobserved local copy"
	x, _ := NewMessage("a", "old")
	if e := m.Receive(context.Background(), x); e != nil {
		t.Fatal(e)
	}
	m.LocalChanged()
	if f.text != "old" || f.writes != 1 {
		t.Fatal("stale hash skipped remote write")
	}
}
