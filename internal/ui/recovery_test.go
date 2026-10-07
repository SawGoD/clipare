package ui

import (
	"testing"
	"time"
)

func TestRecoveryScheduling(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name                       string
		due                        time.Time
		busy, running, setup, want bool
	}{{"not armed", time.Time{}, false, false, false, false}, {"wait", now.Add(time.Second), false, false, false, false}, {"retry", now, false, false, false, true}, {"busy", now, true, false, false, false}, {"running", now, false, true, false, false}, {"setup", now, false, false, true, false}} {
		t.Run(tc.name, func(t *testing.T) {
			if shouldRetry(now, tc.due, tc.busy, tc.running, tc.setup) != tc.want {
				t.Fatal("unexpected retry")
			}
		})
	}
}
