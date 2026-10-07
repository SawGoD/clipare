package transport

import (
	"clipare/internal/config"
	clipsync "clipare/internal/sync"
	"context"
	"log/slog"
	"sync"
	"time"
)

type Workers struct {
	queues []chan clipsync.Message
	wg     sync.WaitGroup
}

func StartWorkers(ctx context.Context, c *Client, peers []config.Peer, log *slog.Logger) *Workers {
	return StartWorkersWithStatus(ctx, c, peers, log, nil)
}
func StartWorkersWithStatus(ctx context.Context, c *Client, peers []config.Peer, log *slog.Logger, status func(string, bool)) *Workers {
	w := &Workers{}
	for _, peer := range peers {
		q := make(chan clipsync.Message, 1)
		w.queues = append(w.queues, q)
		w.wg.Add(1)
		go func(p config.Peer) {
			defer w.wg.Done()
			tick := time.NewTicker(20 * time.Second)
			defer tick.Stop()
			check := func() {
				h, e := c.Health(ctx, p.URL())
				if ctx.Err() != nil {
					return
				}
				log.Debug("peer health", "peer", p.ID, "online", e == nil && h.Device == p.ID)
				if status != nil {
					status(p.ID, e == nil && h.Device == p.ID)
				}
			}
			check()
			for {
				select {
				case <-ctx.Done():
					return
				case m := <-q:
					if ctx.Err() != nil {
						return
					}
					log.Debug("sending clipboard", "peer", p.ID, "id", m.ID)
					if e := c.Send(ctx, p.URL(), m); e != nil && ctx.Err() == nil {
						log.Warn("peer delivery failed", "peer", p.ID, "id", m.ID)
						if status != nil {
							status(p.ID, false)
						}
					}
				case <-tick.C:
					check()
				}
			}
		}(peer)
	}
	return w
}

// Each peer keeps only its newest pending clipboard. No unbounded goroutines or retries.
func (w *Workers) Broadcast(m clipsync.Message) {
	for _, q := range w.queues {
		select {
		case q <- m:
		default:
			select {
			case <-q:
			default:
			}
			select {
			case q <- m:
			default:
			}
		}
	}
}
func (w *Workers) Wait() { w.wg.Wait() }
