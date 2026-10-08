//go:build !windows

package update

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"
)

func configureCommand(cmd *exec.Cmd) {}
func configureHelper(cmd *exec.Cmd)  { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func waitForExit(ctx context.Context, pid int) error {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		if e := syscall.Kill(pid, 0); errors.Is(e, syscall.ESRCH) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
func exitWaiter(pid int) (func(context.Context) error, func(), error) {
	return func(ctx context.Context) error { return waitForExit(ctx, pid) }, func() {}, nil
}
