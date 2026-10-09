//go:build windows

package instance

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestUnresponsiveInstanceNeverStartsSecondGUI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	first, err := Acquire(context.Background(), path)
	if err != nil || !first.Primary {
		t.Fatal("first instance failed", err)
	}
	defer first.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if second, err := Acquire(ctx, path); err == nil || second != nil {
		t.Fatal("second GUI permitted without activation", err)
	}
	first.Close()
	next, err := Acquire(context.Background(), path)
	if err != nil || !next.Primary {
		t.Fatal("exited process still blocks startup", err)
	}
	next.Close()
}
