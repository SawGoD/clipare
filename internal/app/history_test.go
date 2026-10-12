package app

import (
	clipsync "clipare/internal/sync"
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type slowHistory struct {
	release chan struct{}
	started chan struct{}
	count   atomic.Int32
}

func (s *slowHistory) Run(ctx context.Context) { <-ctx.Done() }
func (s *slowHistory) Record(_, _, _ string, _ int64, _ bool) {
	if s.count.Add(1) == 1 {
		close(s.started)
	}
	<-s.release
}

func TestHistoryQueueDoesNotBlockSyncAndWaitsForShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &slowHistory{release: make(chan struct{}), started: make(chan struct{})}
	observe, wait := startHistory(ctx, h, slog.New(slog.NewTextHandler(io.Discard, nil)))
	observe(clipsync.Message{ID: "one"}, true)
	<-h.started
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			observe(clipsync.Message{ID: "next"}, true)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("storage blocked observer")
	}
	cancel()
	stopped := make(chan struct{})
	go func() { wait(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("shutdown failed to wait for writer")
	default:
	}
	close(h.release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("history did not stop")
	}
	if h.count.Load() != 65 {
		t.Fatal("accepted queue not drained", h.count.Load())
	}
	observe(clipsync.Message{ID: "late handler"}, true)
	wait()
}
