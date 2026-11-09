//go:build darwin

package instance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

type Guard struct {
	Primary bool
	file    *os.File
}

func Acquire(ctx context.Context, path string) (*Guard, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(os.TempDir(), Key(path)+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, errors.New("Не удалось проверить запущенный экземпляр Clipare")
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
			return &Guard{Primary: false}, nil
		}
		return nil, errors.New("Не удалось проверить запущенный экземпляр Clipare")
	}
	return &Guard{Primary: true, file: f}, nil
}
func (g *Guard) Close() {
	if g.file != nil {
		_ = syscall.Flock(int(g.file.Fd()), syscall.LOCK_UN)
		_ = g.file.Close()
	}
}
func NotifyFailure(error) {}
