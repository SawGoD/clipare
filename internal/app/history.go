package app

import (
	clipsync "clipare/internal/sync"
	"context"
	"log/slog"
	"sync"
)

// History is best-effort local storage, not part of clipboard delivery.
// A bounded queue keeps filesystem latency outside the sync manager's lock.
func startHistory(ctx context.Context, recorder HistoryRecorder, log *slog.Logger) (func(clipsync.Message, bool), func()) {
	type event struct {
		message clipsync.Message
		local   bool
	}
	queue := make(chan event, 64)
	var queueMu sync.Mutex
	closed := false
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); recorder.Run(ctx) }()
	go func() {
		defer wg.Done()
		write := func(e event) { m := e.message; recorder.Record(m.ID, m.Source, m.Data, m.Timestamp, e.local) }
		for e := range queue {
			write(e)
		}
	}()
	observe := func(m clipsync.Message, local bool) {
		queueMu.Lock()
		defer queueMu.Unlock()
		if closed {
			return
		}
		select {
		case queue <- event{m, local}:
		default:
			log.Warn("history queue full; event skipped")
		}
	}
	// Close only after producers stop, then drain every accepted event.
	return observe, func() {
		queueMu.Lock()
		if !closed {
			closed = true
			close(queue)
		}
		queueMu.Unlock()
		wg.Wait()
	}
}
