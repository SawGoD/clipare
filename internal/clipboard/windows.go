//go:build windows

package clipboard

import (
	"context"
	"errors"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var user = syscall.NewLazyDLL("user32.dll")
var kernel = syscall.NewLazyDLL("kernel32.dll")
var openClipboard = user.NewProc("OpenClipboard")
var closeClipboard = user.NewProc("CloseClipboard")
var getData = user.NewProc("GetClipboardData")
var setData = user.NewProc("SetClipboardData")
var empty = user.NewProc("EmptyClipboard")
var available = user.NewProc("IsClipboardFormatAvailable")
var globalAlloc = kernel.NewProc("GlobalAlloc")
var globalLock = kernel.NewProc("GlobalLock")
var globalUnlock = kernel.NewProc("GlobalUnlock")
var globalFree = kernel.NewProc("GlobalFree")
var globalSize = kernel.NewProc("GlobalSize")
var moveMemory = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlMoveMemory")

type Native struct{}

func New() (Backend, error) { return &Native{}, nil }
func open(owner uintptr) error {
	for i := 0; i < 5; i++ {
		r, _, _ := openClipboard.Call(owner)
		if r != 0 {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("clipboard is busy")
}
func (*Native) Read() (string, bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := open(0); e != nil {
		return "", false, e
	}
	defer closeClipboard.Call()
	ok, _, _ := available.Call(13)
	if ok == 0 {
		return "", false, nil
	}
	h, _, _ := getData.Call(13)
	if h == 0 {
		return "", false, errors.New("clipboard read failed")
	}
	n, _, _ := globalSize.Call(h)
	if n > 2*(maxText+1) {
		return "", false, ErrTooLarge
	}
	if n < 2 {
		return "", false, errors.New("invalid clipboard allocation")
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		return "", false, errors.New("clipboard lock failed")
	}
	defer globalUnlock.Call(h)
	// Copy OS-owned memory into a Go buffer without treating an integer address
	// returned by GlobalLock as a Go pointer.
	buf := make([]uint16, int(n/2))
	moveMemory.Call(uintptr(unsafe.Pointer(&buf[0])), p, uintptr(len(buf)*2))
	end := 0
	for end < len(buf) && buf[end] != 0 {
		end++
	}
	s := string(utf16.Decode(buf[:end]))
	if len(s) > maxText {
		return "", false, ErrTooLarge
	}
	return s, true, nil
}
func (*Native) Write(s string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	b, e := syscall.UTF16FromString(s)
	if e != nil {
		return errors.New("invalid clipboard text")
	}
	// EmptyClipboard must have a real owner for SetClipboardData to succeed.
	class, _ := syscall.UTF16PtrFromString("STATIC")
	owner, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if owner == 0 {
		return errors.New("create clipboard owner failed")
	}
	defer user.NewProc("DestroyWindow").Call(owner)
	if e = open(owner); e != nil {
		return e
	}
	defer closeClipboard.Call()
	h, _, _ := globalAlloc.Call(0x42, uintptr(len(b)*2))
	if h == 0 {
		return errors.New("clipboard allocation failed")
	}
	owned := false
	defer func() {
		if !owned {
			globalFree.Call(h)
		}
	}()
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		return errors.New("clipboard lock failed")
	}
	moveMemory.Call(p, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)*2))
	globalUnlock.Call(h)
	if r, _, _ := empty.Call(); r == 0 {
		return errors.New("clipboard clear failed")
	}
	if r, _, _ := setData.Call(13, h); r == 0 {
		return errors.New("clipboard write failed")
	}
	owned = true
	return nil
}

type wndClass struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	SmallIcon                          uintptr
}
type winMessage struct {
	Window  uintptr
	ID      uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	X, Y    int32
	Private uint32
}

func (*Native) Watch(ctx context.Context, ch chan<- struct{}) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	name, _ := syscall.UTF16PtrFromString("ClipareClipboardListener")
	instance, _, _ := kernel.NewProc("GetModuleHandleW").Call(0)
	callback := syscall.NewCallback(func(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
		if msg == 0x031D {
			notify(ctx, ch)
			return 0
		}
		r, _, _ := user.NewProc("DefWindowProcW").Call(hwnd, uintptr(msg), w, l)
		return r
	})
	wc := wndClass{Proc: callback, Instance: instance, Name: name}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if r, _, _ := user.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return errors.New("register clipboard window failed")
	}
	defer user.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(name)), instance)
	hwnd, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(name)), 0, 0, 0, 0, 0, 0, ^uintptr(2), 0, instance, 0)
	if hwnd == 0 {
		return errors.New("create clipboard window failed")
	}
	defer user.NewProc("DestroyWindow").Call(hwnd)
	if r, _, _ := user.NewProc("AddClipboardFormatListener").Call(hwnd); r == 0 {
		return errors.New("clipboard listener failed")
	}
	defer user.NewProc("RemoveClipboardFormatListener").Call(hwnd)
	// Pump the thread's event queue; clipboard itself is never polled.
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	var m winMessage
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			for {
				r, _, _ := user.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1)
				if r == 0 {
					break
				}
				user.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&m)))
				user.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&m)))
			}
		}
	}
}
