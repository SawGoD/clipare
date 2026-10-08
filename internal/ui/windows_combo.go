//go:build windows

package ui

import "unsafe"

type comboInfo struct {
	Size              uint32
	Item, Button      winRect
	ButtonState       uint32
	Combo, Edit, List uintptr
}

func (n *nativeDesktop) styleCombo(h uintptr) {
	var info comboInfo
	info.Size = uint32(unsafe.Sizeof(info))
	if call("GetComboBoxInfo", h, uintptr(unsafe.Pointer(&info))) == 0 {
		return
	}
	// The editable child is still native; remove its independent sunken frame.
	style := call("GetWindowLongPtrW", info.Edit, ^uintptr(15))
	call("SetWindowLongPtrW", info.Edit, ^uintptr(15), style&^0x00800000)
	extended := call("GetWindowLongPtrW", info.Edit, ^uintptr(19))
	call("SetWindowLongPtrW", info.Edit, ^uintptr(19), extended&^0x200)
	call("SetWindowPos", info.Edit, 0, 0, 0, 0, 0, 0x37)
	call("SendMessageW", h, 0x153, ^uintptr(0), uintptr(n.scaledRect(n.window, logicalRect{h: 26}).Bottom))
	call("SendMessageW", h, 0x153, 0, uintptr(n.scaledRect(n.window, logicalRect{h: 32}).Bottom))
	n.controls[info.List] = winControl{class: "COMBOLIST"}
	n.styleControl(info.List)
	// The popup keeps CBS_HASSTRINGS and native selection/accessibility. Its rows
	// are painted by the owner's WM_DRAWITEM handler, not the classic grey theme.
}

func (n *nativeDesktop) comboOpen() bool {
	return call("SendMessageW", n.fields[2], 0x157, 0, 0) != 0
}
