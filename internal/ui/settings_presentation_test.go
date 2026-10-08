package ui

import "testing"

func TestSettingsPresentation(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		for _, expanded := range []bool{false, true} {
			s := presentSettings(count, expanded)
			if s.EmptyDevices != (count == 0) || s.ShowPeerList != (count > 0) || s.ShowRemove != (count > 0) || s.ShowCompactAdd != (count > 0) || s.ShowPreferences != expanded {
				t.Fatalf("count=%d expanded=%v: %+v", count, expanded, s)
			}
		}
	}
}
