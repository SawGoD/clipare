package clipboard

import (
	"context"
	"errors"
)

var ErrTooLarge = errors.New("clipboard text exceeds 1 MiB")

const maxText = 1 << 20

// Watch reports invalidations, never queued text snapshots. Consumers read under
// the same lock used for remote writes, preventing stale events from echoing.
type Backend interface {
	Read() (string, bool, error)
	Write(string) error
	Watch(context.Context, chan<- struct{}) error
}

func notify(ctx context.Context, ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	case <-ctx.Done():
	default:
	}
}
