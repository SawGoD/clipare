//go:build darwin && cgo

package ui

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#include "native_darwin.h"
*/
import "C"
import (
	"clipare/internal/config"
	"strings"
	"unsafe"
)

type nativeDesktop struct{}

func newDesktop() (desktop, error)    { return &nativeDesktop{}, nil }
func (*nativeDesktop) Init() error    { C.clipare_init(); return nil }
func (*nativeDesktop) Poll() int      { return int(C.clipare_poll()) }
func cstr(s string, fn func(*C.char)) { p := C.CString(s); defer C.free(unsafe.Pointer(p)); fn(p) }
func (*nativeDesktop) Show(f form, peers []config.Peer, addresses []string) {
	cstr(strings.Join(addresses, "\n"), func(p *C.char) { C.clipare_addresses(p) })
	for i, v := range f.Values {
		cstr(v, func(p *C.char) { C.clipare_set(C.int(i), p) })
	}
	var names []string
	for _, p := range peers {
		names = append(names, peerLabel(p))
	}
	cstr(strings.Join(names, "\n"), func(p *C.char) { C.clipare_peers(p) })
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
	n := 0
	if enabled {
		n = 1
	}
	cstr(status, func(s *C.char) {
		cstr(peerStatuses(peers, states), func(p *C.char) { C.clipare_status(s, C.int(n), p) })
	})
}
func (*nativeDesktop) Alert(s string) { cstr(s, func(p *C.char) { C.clipare_alert(p) }) }
func (*nativeDesktop) Close()         { C.clipare_close() }
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
func (*nativeDesktop) UpdatePrompt(text string, installable bool) {
	n := 0
	if installable {
		n = 1
	}
	cstr(text, func(p *C.char) { C.clipare_update_prompt(p, C.int(n)) })
}
func (*nativeDesktop) UpdateClose() { C.clipare_update_close() }
