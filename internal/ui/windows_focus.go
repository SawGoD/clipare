//go:build windows

package ui

// Follow native UI cues: mouse input keeps decoration quiet; keyboard
// navigation must still have a visible focus indicator (also in Contrast mode).
func (n *nativeDesktop) keyboardFocusVisible() bool {
	return call("SendMessageW", n.main, 0x129, 0, 0)&1 == 0 // WM_QUERYUISTATE, UISF_HIDEFOCUS
}

func (n *nativeDesktop) focusCues(keyboard bool) {
	if n.keyboardFocusVisible() == keyboard {
		return
	}
	action := uintptr(1) // UIS_SET
	if keyboard {
		action = 2
	} // UIS_CLEAR
	call("SendMessageW", n.main, 0x127, action|(1<<16), 0) // WM_CHANGEUISTATE
	call("RedrawWindow", n.activePanel(), 0, 0, 0x81)      // invalidate descendants, no erase
}
