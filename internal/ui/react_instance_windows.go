//go:build windows

package ui

import (
	"clipare/internal/instance"
	"errors"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const gwlpReactSubclassID = 0x434c4950

func installReactActivation(window *application.WebviewWindow, path string) (func(), error) {
	hwnd := uintptr(window.NativeWindow())
	if hwnd == 0 {
		return nil, errors.New("window handle unavailable")
	}
	key := instance.Key(path)
	w, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return nil, err
	}
	user := syscall.NewLazyDLL("user32.dll")
	comctl := syscall.NewLazyDLL("comctl32.dll")
	message, _, _ := user.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(w)))
	if message == 0 {
		return nil, errors.New("activation message unavailable")
	}
	callback := syscall.NewCallback(func(h, msg, wp, lp, id, ref uintptr) uintptr {
		if msg == message {
			go func() { window.Show(); window.Focus() }()
			return 1
		}
		result, _, _ := comctl.NewProc("DefSubclassProc").Call(h, msg, wp, lp)
		return result
	})
	ok, _, _ := comctl.NewProc("SetWindowSubclass").Call(hwnd, callback, gwlpReactSubclassID, 0)
	if ok == 0 {
		return nil, errors.New("cannot register activation handler")
	}
	ok, _, _ = user.NewProc("SetPropW").Call(hwnd, uintptr(unsafe.Pointer(w)), 1)
	if ok == 0 {
		comctl.NewProc("RemoveWindowSubclass").Call(hwnd, callback, gwlpReactSubclassID)
		return nil, errors.New("cannot publish activation window")
	}
	return func() {
		user.NewProc("RemovePropW").Call(hwnd, uintptr(unsafe.Pointer(w)))
		comctl.NewProc("RemoveWindowSubclass").Call(hwnd, callback, gwlpReactSubclassID)
	}, nil
}
