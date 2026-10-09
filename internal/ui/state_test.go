package ui

import "testing"

type preferenceRecorder struct {
	desktop
	autostart, updates bool
}

func (p *preferenceRecorder) Preferences(autostart, updates bool) {
	p.autostart, p.updates = autostart, updates
}
func TestRestorePreferencesWithoutReplacingDraftOrView(t *testing.T) {
	p := &preferenceRecorder{}
	// The nil desktop intentionally fails if restoration calls Read/Show.
	for _, values := range [][2]bool{{true, false}, {false, true}, {false, false}} {
		renderPreferences(p, values[0], values[1])
		if p.autostart != values[0] || p.updates != values[1] {
			t.Fatal("preferences not restored")
		}
	}
}

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

func TestDiscoverySharesHomeContainer(t *testing.T) {
	n := Navigation{AdditionalExpanded: true, AdvancedExpanded: true}
	n.Open(ViewDiscovery)
	if n.Container() != ViewHome || n.View != ViewDiscovery {
		t.Fatal("discovery replaced home")
	}
	n.Home()
	if !n.AdditionalExpanded || !n.AdvancedExpanded {
		t.Fatal("inline discovery lost settings")
	}
	n.Open(ViewPairing)
	if n.Container() != ViewHome || n.View != ViewPairing {
		t.Fatal("pairing replaced home")
	}
}

func TestInlineNoticeReturnsToActiveWorkflow(t *testing.T) {
	for _, view := range []ViewState{ViewHome, ViewDiscovery, ViewPairing, ViewUpdate} {
		n := Navigation{View: view}
		n.Open(ViewNotice)
		n.Open(ViewNotice)
		if n.DismissNotice() != view {
			t.Fatal("notice lost the active workflow")
		}
		n.Open(ViewNotice)
		n.End(view)
		if n.DismissNotice() != ViewHome {
			t.Fatal("notice returned to a completed workflow")
		}
	}
}
