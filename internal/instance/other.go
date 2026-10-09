//go:build !windows

package instance

import "context"

type Guard struct{ Primary bool }

func Acquire(context.Context, string) (*Guard, error) { return &Guard{Primary: true}, nil }
func (*Guard) Close()                                 {}
func NotifyFailure(error)                             {}
