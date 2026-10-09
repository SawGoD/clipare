//go:build windows

package instance

import (
	"context"
	"errors"
	"syscall"
	"time"
	"unsafe"
)

var kernel = syscall.NewLazyDLL("kernel32.dll")
var user = syscall.NewLazyDLL("user32.dll")

type Guard struct {
	Primary bool
	handle  uintptr
}

func (g *Guard) Close() {
	if g.handle != 0 {
		kernel.NewProc("CloseHandle").Call(g.handle)
		g.handle = 0
	}
}
func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

// The GUI executable has no console. Activation failures must not disappear
// into stderr or start a second application listening on the same address.
func NotifyFailure(err error) {
	user.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(utf16(err.Error()))), uintptr(unsafe.Pointer(utf16("Clipare"))), 0x10)
}

// An existing GUI owns the kernel object's lifetime, even if its window is
// hidden or still starting. Do not start another listener if activation fails.
func Acquire(ctx context.Context, path string) (*Guard, error) {
	key := Key(path)
	h, _, err := kernel.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(utf16(`Local\`+key))))
	if h == 0 {
		return nil, errors.New("Не удалось проверить запущенный экземпляр Clipare")
	}
	g := &Guard{handle: h, Primary: err != syscall.Errno(183)}
	if g.Primary {
		return g, nil
	}
	defer g.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	msg, _, _ := user.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(utf16(key))))
	var target uintptr
	callback := syscall.NewCallback(func(hwnd, param uintptr) uintptr {
		property, _, _ := user.NewProc("GetPropW").Call(hwnd, uintptr(unsafe.Pointer(utf16(key))))
		if property != 0 {
			target = hwnd
			return 0
		}
		return 1
	})
	for {
		user.NewProc("EnumWindows").Call(callback, 0)
		if target != 0 {
			var acknowledged uintptr
			ok, _, _ := user.NewProc("SendMessageTimeoutW").Call(target, msg, 0, 0, 3, 500, uintptr(unsafe.Pointer(&acknowledged)))
			if ok != 0 && acknowledged == 1 {
				return &Guard{}, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, errors.New("Clipare уже запущен, но не отвечает на открытие окна. Второй экземпляр не запущен")
		case <-time.After(100 * time.Millisecond):
		}
	}
}
