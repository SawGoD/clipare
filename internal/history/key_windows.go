//go:build windows

package history

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var crypt32 = syscall.NewLazyDLL("crypt32.dll")
var k32 = syscall.NewLazyDLL("kernel32.dll")

// dataBlob mirrors DATA_BLOB for DPAPI. The pointer is typed so the result
// buffer can be read back without uintptr conversions.
type dataBlob struct {
	Size uint32
	Data *byte
}

// dpapiKeystore stores the master key DPAPI-encrypted (user-bound) in a small
// file. The file alone is useless without the current Windows user profile.
type dpapiKeystore struct{ path string }

func (k dpapiKeystore) Load() ([]byte, error) {
	blob, e := os.ReadFile(k.path)
	if errors.Is(e, fs.ErrNotExist) {
		return nil, os.ErrNotExist
	}
	if e != nil {
		return nil, errors.New("cannot read history key file")
	}
	key, e := unprotect(blob)
	if e != nil {
		return nil, e
	}
	if len(key) != keySize {
		return nil, errors.New("history key has unexpected size")
	}
	return key, nil
}

func (k dpapiKeystore) Save(key []byte) error {
	if len(key) != keySize {
		return errors.New("history key must contain 32 bytes")
	}
	blob, e := protect(key)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(k.path), 0700); e != nil {
		return errors.New("cannot create history key directory")
	}
	if e = os.WriteFile(k.path, blob, 0600); e != nil {
		return errors.New("cannot write history key file")
	}
	return nil
}

func protect(data []byte) ([]byte, error) {
	in := dataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out dataBlob
	r, _, _ := crypt32.NewProc("CryptProtectData").Call(
		uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, errors.New("DPAPI protection failed")
	}
	defer localFree(&out)
	return blobBytes(out), nil
}

func unprotect(blob []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, errors.New("history key file is empty")
	}
	in := dataBlob{Size: uint32(len(blob)), Data: &blob[0]}
	var out dataBlob
	r, _, _ := crypt32.NewProc("CryptUnprotectData").Call(
		uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, errors.New("history key cannot be decrypted with this user profile")
	}
	defer localFree(&out)
	return blobBytes(out), nil
}

func blobBytes(b dataBlob) []byte {
	if b.Size == 0 || b.Data == nil {
		return nil
	}
	out := make([]byte, b.Size)
	copy(out, unsafe.Slice(b.Data, b.Size))
	return out
}

func localFree(b *dataBlob) {
	if b.Data != nil {
		k32.NewProc("LocalFree").Call(uintptr(unsafe.Pointer(b.Data)))
		b.Data = nil
	}
}

// DefaultKeyPath returns the DPAPI-protected key file location next to the
// user configuration, never inside the history database directory itself.
func DefaultKeyPath() (string, error) {
	base, e := os.UserConfigDir()
	if e != nil {
		return "", fmt.Errorf("history key location unavailable: %w", e)
	}
	return filepath.Join(base, "Clipare", "history.key"), nil
}

func init() {
	defaultKeystore = func() Keystore {
		path, e := DefaultKeyPath()
		if e != nil {
			return nil
		}
		return dpapiKeystore{path: path}
	}()
}
