//go:build windows

package app

import (
	"errors"
	"net"
	"syscall"
	"testing"
)

func TestWinsockAddressUnavailable(t *testing.T) {
	if !errors.Is(listenError(&net.OpError{Op: "listen", Err: syscall.Errno(10049)}), ErrAddressUnavailable) {
		t.Fatal("real Winsock error not recoverable")
	}
	if errors.Is(listenError(syscall.Errno(10048)), ErrAddressUnavailable) {
		t.Fatal("occupied Windows port incorrectly recoverable")
	}
	listener, err := net.Listen("tcp", "192.0.2.123:0")
	if err == nil {
		listener.Close()
		t.Skip("documentation address configured on test host")
	}
	if !errors.Is(listenError(err), ErrAddressUnavailable) {
		t.Fatal("native Windows bind error not recognized", err)
	}
}
