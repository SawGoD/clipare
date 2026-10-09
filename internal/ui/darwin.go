//go:build darwin && cgo

package ui

/*
#cgo LDFLAGS: -framework AppKit -framework UserNotifications
#include <stdlib.h>
#include "native_darwin.h"
*/
import "C"
import (
	"clipare/internal/config"
	"encoding/json"
	"strings"
	"unsafe"
)

type nativeDesktop struct{ navigation Navigation }

func newDesktop() (desktop, error) { return &nativeDesktop{}, nil }
func (*nativeDesktop) Init() error { C.clipare_init(); return nil }
func (n *nativeDesktop) Poll() int {
	event := int(C.clipare_poll())
	n.navigation.View = ViewState(C.clipare_view())
	disclosures := int(C.clipare_disclosures())
	n.navigation.AdditionalExpanded = disclosures&1 != 0
	n.navigation.AdvancedExpanded = disclosures&2 != 0
	return event
}
func cstr(s string, fn func(*C.char)) { p := C.CString(s); defer C.free(unsafe.Pointer(p)); fn(p) }
func (*nativeDesktop) Show(f form, peers []config.Peer, addresses []string) {
	cstr(strings.Join(addresses, "\n"), func(p *C.char) { C.clipare_addresses(p) })
	for i, v := range f.Values {
		cstr(v, func(p *C.char) { C.clipare_set(C.int(i), p) })
	}
	renderDarwinRows(peers, nil)
	auto := 0
	if f.Autostart {
		auto = 1
	}
	C.clipare_auto(C.int(auto))
	C.clipare_show()
}
func (*nativeDesktop) Read() form {
	var f form
	for i := range f.Values {
		p := C.clipare_get(C.int(i))
		f.Values[i] = C.GoString(p)
		C.free(unsafe.Pointer(p))
	}
	f.Autostart = C.clipare_is_auto() != 0
	return f
}
func (*nativeDesktop) Selected() int { return int(C.clipare_selected()) }
func (*nativeDesktop) SetPeer(p config.Peer) {
	for i, v := range peerValues(p) {
		cstr(v, func(s *C.char) { C.clipare_set(C.int(i+6), s) })
	}
}
func (*nativeDesktop) Update(status string, enabled bool, peers []config.Peer, states map[string]bool) {
	renderDarwinRows(peers, states)
}
func renderDarwinRows(peers []config.Peer, states map[string]bool) {
	rows := make([]map[string]any, 0, len(peers))
	for _, row := range pairedRows(peers, states) {
		rows = append(rows, map[string]any{"name": row.name, "detail": row.detail, "online": row.online})
	}
	data, _ := json.Marshal(rows)
	s := presentSettings(len(peers), false)
	flag := func(value bool) C.int {
		if value {
			return 1
		}
		return 0
	}
	cstr(string(data), func(p *C.char) {
		C.clipare_device_rows(p, flag(s.EmptyDevices), flag(s.ShowPeerList), flag(s.ShowRemove), flag(s.ShowCompactAdd))
	})
}
func (*nativeDesktop) SyncStatus(s SyncStatus) {
	cstr(s.Message, func(m *C.char) { cstr(s.Reason, func(r *C.char) { C.clipare_sync_status(C.int(s.State), m, r) }) })
}
func (*nativeDesktop) Alert(s string) { cstr(s, func(p *C.char) { C.clipare_alert(p) }) }
func (*nativeDesktop) Preferences(autostart, updates bool) {
	flag := func(v bool) C.int {
		if v {
			return 1
		}
		return 0
	}
	C.clipare_preferences(flag(autostart), flag(updates))
}
func (*nativeDesktop) Close() { C.clipare_close() }
func (*nativeDesktop) Discovered(lines, status string) {
	cstr(lines, func(l *C.char) { cstr(status, func(s *C.char) { C.clipare_discovered(l, s) }) })
}
func (*nativeDesktop) DiscoveredSelected() int { return int(C.clipare_discovered_selected()) }
func (*nativeDesktop) Pair(name, sas string, mode int) {
	n := mode
	cstr(name, func(a *C.char) { cstr(sas, func(b *C.char) { C.clipare_pair(a, b, C.int(n)) }) })
}
func (*nativeDesktop) PairClose() { C.clipare_pair_close() }
func (*nativeDesktop) UpdateSettings(version string, enabled bool) {
	n := 0
	if enabled {
		n = 1
	}
	cstr(version, func(p *C.char) { C.clipare_update_settings(p, C.int(n)) })
}
func (*nativeDesktop) UpdateEnabled() bool { return C.clipare_update_enabled() != 0 }
func (*nativeDesktop) UpdatePrompt(v updatePrompt) {
	cstr(v.Text, func(t *C.char) {
		cstr(v.Primary, func(p *C.char) {
			cstr(v.Dismiss, func(d *C.char) { C.clipare_update_prompt(t, p, d, C.int(v.Action)) })
		})
	})
}
func (*nativeDesktop) UpdateClose() { C.clipare_update_close() }
