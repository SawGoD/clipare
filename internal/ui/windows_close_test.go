//go:build windows

package ui

import "testing"

func TestAltCloseCommand(t *testing.T) {
	cases := []struct {
		name    string
		msg     uint32
		w       uintptr
		altDown bool
		want    bool
	}{
		{"alt f4", wmSysCommand, scClose, true, true},
		{"alt f4 ignores low bits", wmSysCommand, scClose | 0x3, true, true},
		{"caption close button", wmSysCommand, scClose, false, false},
		{"other system command", wmSysCommand, 0xf030, true, false},
		{"wm close without alt", 0x10, 0, true, false},
	}
	for _, c := range cases {
		if got := altCloseCommand(c.msg, c.w, c.altDown); got != c.want {
			t.Errorf("%s: altCloseCommand(%#x, %#x, %v) = %v, want %v", c.name, c.msg, c.w, c.altDown, got, c.want)
		}
	}
}
