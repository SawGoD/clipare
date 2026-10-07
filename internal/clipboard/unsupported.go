//go:build (!windows && !darwin) || (darwin && !cgo)

package clipboard

import "errors"

func New() (Backend, error) {
	return nil, errors.New("clipboard requires Windows or macOS with CGO enabled")
}
