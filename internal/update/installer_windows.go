package update

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

func configureCommand(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} }
func configureHelper(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x208}
}
func waitForExit(ctx context.Context, pid int) error {
	handle, e := syscall.OpenProcess(0x100000, false, uint32(pid))
	if e != nil {
		return e
	}
	defer syscall.CloseHandle(handle)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		r, e := syscall.WaitForSingleObject(handle, 0)
		if e != nil {
			return e
		}
		if r == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
