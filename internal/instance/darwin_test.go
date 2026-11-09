//go:build darwin

package instance

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAcquireExclusiveProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	first, err := Acquire(context.Background(), path)
	if err != nil || !first.Primary {
		t.Fatalf("first: %v", err)
	}
	second, err := Acquire(context.Background(), path)
	if err != nil || second.Primary {
		t.Fatalf("second: %+v %v", second, err)
	}
	first.Close()
	third, err := Acquire(context.Background(), path)
	if err != nil || !third.Primary {
		t.Fatalf("after close: %v", err)
	}
	third.Close()
}
