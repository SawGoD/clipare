package ui

import (
	"clipare/internal/app"
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestRemoveIgnoresTechnicalFormAndPreservesOtherConfig(t *testing.T) {
	c, _ := config.Default()
	c.Peers = []config.Peer{{ID: "a", PublicKey: "key-a"}, {ID: "b", Legacy: true}}
	beforePort := c.Listen.Port
	next := removeDevice(c, "a")
	if next.Listen.Port != beforePort || len(next.Peers) != 1 || next.Peers[0].ID != "b" || !reflect.DeepEqual(next.Removed, []string{"a"}) || len(c.Peers) != 2 || len(c.Removed) != 0 {
		t.Fatal("removal changed unrelated configuration")
	}
	if _, err := fromForm(form{}, c); err == nil {
		t.Fatal("fixture must represent an incomplete draft")
	}
	if len(removeDevice(next, "b").Peers) != 0 {
		t.Fatal("legacy deletion failed")
	}
}

type removalDesktop struct {
	*testPairDesktop
	phase int
}

func (*removalDesktop) Selected() int { return 0 }
func (d *removalDesktop) Update(_ string, enabled bool, peers []config.Peer, _ map[string]bool) {
	if !enabled {
		return
	}
	if d.phase == 0 {
		d.phase = 1
		d.f.Values[3] = ""
		d.actions <- eventRemove
		return
	}
	if d.phase == 1 && len(peers) == 0 {
		d.phase = 2
		d.actions <- eventQuit
	}
}
func TestRemovePeerWithEmptyPortPersistsThroughGUILifecycle(t *testing.T) {
	c, _ := config.Default()
	c.Listen.Address = "127.0.0.1"
	c.Listen.Port = 45873
	c.Peers = []config.Peer{{ID: "desktop", Name: "Desktop", Address: "100.64.0.2", Port: 45873, Legacy: true, LegacySecret: c.Security.Secret}}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.Save(path, c); err != nil {
		t.Fatal(err)
	}
	d := &removalDesktop{testPairDesktop: &testPairDesktop{&testDesktop{actions: make(chan int, 8), path: path}}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := func(ctx context.Context, c config.Config, b clipboard.Backend, log *slog.Logger, h func(string, bool)) (*app.Session, error) {
		c.Listen.Port = 0
		return app.Start(ctx, c, b, log, h)
	}
	if err := runDesktop(ctx, path, slog.New(slog.NewTextHandler(io.Discard, nil)), d, testClipboard{}, start, loopTiming{time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	saved, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if d.phase != 2 || len(d.alerts) != 0 || len(saved.Peers) != 0 || saved.Listen.Port != 45873 {
		t.Fatalf("removal failed: phase=%d peers=%d port=%d alerts=%v", d.phase, len(saved.Peers), saved.Listen.Port, d.alerts)
	}
}
func TestEmptyInputsDisableActions(t *testing.T) {
	for _, action := range []int{eventSave, eventImport, eventUpsert, eventCopy} {
		if actionReady(action, form{}) {
			t.Fatal("empty form accepted")
		}
	}
	f := form{}
	f.Values[10] = "  \n"
	if actionReady(eventImport, f) {
		t.Fatal("whitespace enabled import")
	}
	f.Values[10] = "clipare1:example"
	if !actionReady(eventImport, f) {
		t.Fatal("nonempty import disabled")
	}
}

func TestApplyVisibilityTracksOnlyRelatedChanges(t *testing.T) {
	saved := form{Values: [11]string{"MacBook", "id", "100.64.0.1", "45873", "test-secret"}}
	if applyMask(saved, saved, false) != 0 {
		t.Fatal("unchanged form shows apply")
	}
	for _, index := range []int{0, 2, 3, 4} {
		current := saved
		current.Values[index] += "x"
		want := uint64(1 << eventSave)
		if index == 0 {
			want |= 1 << eventDeviceName
		}
		if got := applyMask(current, saved, false); got != want {
			t.Fatalf("field %d: got %x want %x", index, got, want)
		}
		if applyMask(current, current, false) != 0 {
			t.Fatal("successful save retains apply")
		}
		current.Values[index] = saved.Values[index]
		if applyMask(current, saved, false) != 0 {
			t.Fatal("reverted edit retains apply")
		}
	}
	for _, index := range []int{1, 5, 6, 7, 8, 9, 10} {
		current := saved
		current.Values[index] += "x"
		if applyMask(current, saved, false) != 0 {
			t.Fatalf("unrelated field %d shows apply", index)
		}
	}
	if applyMask(saved, saved, true) != 1<<eventSave {
		t.Fatal("pending legacy peer/setup cannot be saved")
	}
}
