//go:build windows

package clipboard

import (
	"reflect"
	"syscall"
	"testing"
	"unsafe"
)

// Exercise the real WinAPI memory boundary without touching the clipboard.
func TestGlobalMemoryCopy(t *testing.T) {
	source, _ := syscall.UTF16FromString("Clipare 世界\n")
	size := uintptr(len(source) * 2)
	handle, _, _ := globalAlloc.Call(0x42, size)
	if handle == 0 {
		t.Fatal("GlobalAlloc failed")
	}
	defer globalFree.Call(handle)
	address, _, _ := globalLock.Call(handle)
	if address == 0 {
		t.Fatal("GlobalLock failed")
	}
	defer globalUnlock.Call(handle)
	moveMemory.Call(address, uintptr(unsafe.Pointer(&source[0])), size)
	dest := make([]uint16, len(source))
	moveMemory.Call(uintptr(unsafe.Pointer(&dest[0])), address, size)
	if !reflect.DeepEqual(source, dest) {
		t.Fatal("WinAPI memory copy corrupted UTF-16")
	}
}
