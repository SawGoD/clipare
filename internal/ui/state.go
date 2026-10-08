package ui

type ViewState int

const (
	ViewHome ViewState = iota
	ViewDiscovery
	ViewPairing
	ViewUpdate
	ViewNotice
)

type Navigation struct {
	View                                 ViewState
	AdditionalExpanded, AdvancedExpanded bool
}

func (n *Navigation) Open(v ViewState) { n.View = v }
func (n *Navigation) Home()            { n.View = ViewHome }

type SyncState int

const (
	SyncActive SyncState = iota
	SyncDisabled
	SyncDegraded
)

type SyncStatus struct {
	State           SyncState
	Message, Reason string
}

func (s SyncStatus) Color() uint32 {
	switch s.State {
	case SyncActive:
		return 0x32754c
	case SyncDisabled:
		return 0xc4313b
	default:
		return 0xc77600
	}
}

func syncStatus(requested, running, localOnly, applying bool, reason string) SyncStatus {
	if !requested {
		return SyncStatus{State: SyncDisabled, Message: "Синхронизация приостановлена"}
	}
	if !running || localOnly || applying {
		return SyncStatus{State: SyncDegraded, Message: "Синхронизация недоступна", Reason: reason}
	}
	return SyncStatus{State: SyncActive, Message: "Синхронизация включена"}
}

type statusDesktop interface{ SyncStatus(SyncStatus) }

// Preserve the diagnostic string contract for existing headless GUI-loop tests;
// native adapters receive typed state sourced from the actual session lifecycle.
func renderStatus(d desktop, reason string, requested, running, localOnly, applying bool) {
	if native, ok := d.(statusDesktop); ok {
		native.SyncStatus(syncStatus(requested, running, localOnly, applying, reason))
	}
}
