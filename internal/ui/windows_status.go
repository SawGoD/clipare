//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

func (n *nativeDesktop) SyncStatus(s SyncStatus) {
	previous := n.utilityStatus
	if previous == s {
		return
	}
	n.utilityStatus = s
	n.enabled = s.State != SyncDisabled
	n.state = s.Message
	if s.Reason != "" {
		n.state += " — " + s.Reason
	}
	call("SetWindowTextW", n.homeStatus, uintptr(unsafe.Pointer(wide(n.state))))
	call("InvalidateRect", n.statusDot, 0, 1)
	if n.statusIcons[s.State] == 0 {
		n.statusIcons[s.State] = statusIcon(s)
	}
	if n.statusIcons[s.State] != 0 {
		n.icon = n.statusIcons[s.State]
	}
	data := notifyIcon{Window: n.main, ID: 1, Flags: 3, Icon: n.icon}
	data.Size = uint32(unsafe.Sizeof(data))
	copy(data.Tip[:], syscall.StringToUTF16("Clipare — "+n.state))
	data.Tip[len(data.Tip)-1] = 0
	shell.NewProc("Shell_NotifyIconW").Call(1, uintptr(unsafe.Pointer(&data)))
}

// Compose one Clipare clipboard mark and a state badge, not unrelated bitmaps.
func statusIcon(s SyncStatus) uintptr {
	var pixels [32 * 32 * 4]byte
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			var rgb uint32
			opaque := false
			if x >= 7 && x <= 23 && y >= 6 && y <= 26 {
				rgb = 0x626e79
				opaque = true
			}
			if x >= 9 && x <= 21 && y >= 8 && y <= 24 {
				rgb = 0xf3f5f7
			}
			if x >= 11 && x <= 19 && y >= 3 && y <= 8 {
				rgb = 0x626e79
				opaque = true
			}
			if x >= 11 && x <= 19 && (y == 12 || y == 16 || y == 20) {
				rgb = 0x626e79
			}
			d := (x-24)*(x-24) + (y-24)*(y-24)
			if d <= 49 {
				rgb = 0xffffff
				opaque = true
			}
			if d <= 36 {
				rgb = s.Color()
			}
			if opaque {
				i := (y*32 + x) * 4
				pixels[i] = byte(rgb)
				pixels[i+1] = byte(rgb >> 8)
				pixels[i+2] = byte(rgb >> 16)
				pixels[i+3] = 255
			}
		}
	}
	var bitmap struct {
		Size                   uint32
		Width, Height          int32
		Planes, Bits           uint16
		Compression, ImageSize uint32
		XPels, YPels           int32
		Used, Important        uint32
	}
	bitmap.Size = 40
	bitmap.Width = 32
	bitmap.Height = -32
	bitmap.Planes = 1
	bitmap.Bits = 32
	var bits uintptr
	image := gcall("CreateDIBSection", 0, uintptr(unsafe.Pointer(&bitmap)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if image == 0 {
		return 0
	}
	defer gcall("DeleteObject", image)
	k32.NewProc("RtlMoveMemory").Call(bits, uintptr(unsafe.Pointer(&pixels[0])), uintptr(len(pixels)))
	var mask [128]byte
	alphaMask := gcall("CreateBitmap", 32, 32, 1, 1, uintptr(unsafe.Pointer(&mask[0])))
	if alphaMask == 0 {
		return 0
	}
	defer gcall("DeleteObject", alphaMask)
	info := struct {
		IsIcon      int32
		X, Y        uint32
		Mask, Color uintptr
	}{IsIcon: 1, Mask: alphaMask, Color: image}
	return call("CreateIconIndirect", uintptr(unsafe.Pointer(&info)))
}

func (n *nativeDesktop) menuStatusIcon(menu uintptr) uintptr {
	icon := call("CopyImage", n.icon, 1, 16, 16, 0)
	if icon == 0 {
		return 0
	}
	defer call("DestroyIcon", icon)
	var image struct {
		IsIcon      int32
		X, Y        uint32
		Mask, Color uintptr
	}
	if call("GetIconInfo", icon, uintptr(unsafe.Pointer(&image))) == 0 {
		return 0
	}
	if image.Mask != 0 {
		gcall("DeleteObject", image.Mask)
	}
	var item struct {
		Size, Mask, Type, State, ID             uint32
		Submenu, Checked, Unchecked, Data, Text uintptr
		Length                                  uint32
		Bitmap                                  uintptr
	}
	item.Size = uint32(unsafe.Sizeof(item))
	item.Mask = 0x80
	item.Bitmap = image.Color
	call("SetMenuItemInfoW", menu, 1, 1, uintptr(unsafe.Pointer(&item)))
	return image.Color
}

func (n *nativeDesktop) notifyPair(name string) {
	data := notifyIcon{Window: n.main, ID: 1, Flags: 16, InfoFlags: 1}
	data.Size = uint32(unsafe.Sizeof(data))
	copy(data.InfoTitle[:], syscall.StringToUTF16("Новое устройство хочет подключиться"))
	copy(data.Info[:], syscall.StringToUTF16(name))
	data.Info[len(data.Info)-1] = 0
	shell.NewProc("Shell_NotifyIconW").Call(1, uintptr(unsafe.Pointer(&data)))
}
