//go:build windows

package autostart

import (
	"errors"
	"syscall"
	"unsafe"
)

func Set(enabled bool, executable, configPath string) error {
	dll := syscall.NewLazyDLL("advapi32.dll")
	sub, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var key uintptr
	r, _, _ := dll.NewProc("RegCreateKeyExW").Call(0x80000001, uintptr(unsafe.Pointer(sub)), 0, 0, 0, 0x0002, 0, uintptr(unsafe.Pointer(&key)), 0)
	if r != 0 {
		return errors.New("Не удалось открыть настройки автозапуска Windows")
	}
	defer dll.NewProc("RegCloseKey").Call(key)
	name, _ := syscall.UTF16PtrFromString("Clipare")
	if !enabled {
		r, _, _ = dll.NewProc("RegDeleteValueW").Call(key, uintptr(unsafe.Pointer(name)))
		if r == 0 || r == 2 {
			return nil
		}
		return errors.New("Не удалось отключить автозапуск")
	}
	value, _ := syscall.UTF16FromString(syscall.EscapeArg(executable) + " --config " + syscall.EscapeArg(configPath))
	r, _, _ = dll.NewProc("RegSetValueExW").Call(key, uintptr(unsafe.Pointer(name)), 0, 1, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)*2))
	if r != 0 {
		return errors.New("Не удалось сохранить автозапуск")
	}
	return nil
}
