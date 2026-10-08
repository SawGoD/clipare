package ui

import "testing"

func TestSyncState(t *testing.T) {
	for _, test := range []struct {
		requested, running, local, applying bool
		want                                SyncState
		color                               uint32
	}{
		{true, true, false, false, SyncActive, 0x32754c},
		{false, true, false, false, SyncDisabled, 0xc4313b},
		{true, false, false, false, SyncDegraded, 0xc77600},
		{true, true, true, false, SyncDegraded, 0xc77600},
		{true, true, false, true, SyncDegraded, 0xc77600},
	} {
		s := syncStatus(test.requested, test.running, test.local, test.applying, "Ожидание NetBird")
		if s.State != test.want || s.Color() != test.color || s.Message == "" {
			t.Fatalf("%+v: %+v", test, s)
		}
	}
	// Peer state is deliberately absent from this contract.
	peers := map[string]bool{"desktop": true, "laptop": false}
	_ = peers
	if syncStatus(true, true, false, false, "").State != SyncActive {
		t.Fatal("offline peer changed utility state")
	}
}

func TestNavigation(t *testing.T) {
	n := Navigation{}
	for _, view := range []ViewState{ViewDiscovery, ViewPairing, ViewHome, ViewUpdate, ViewHome} {
		n.Open(view)
		if n.View != view {
			t.Fatal("navigation failed")
		}
	}
	n.AdvancedExpanded = true
	n.Open(ViewDiscovery)
	n.Home()
	if !n.AdvancedExpanded || n.View != ViewHome {
		t.Fatal("navigation lost disclosure state")
	}
}
