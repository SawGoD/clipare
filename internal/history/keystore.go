package history

import "errors"

// Keystore persists the history master key with OS user-bound protection.
// The key never touches config files, the history file itself, or logs.
type Keystore interface {
	// Load returns the stored key or os.ErrNotExist when none exists yet.
	Load() ([]byte, error)
	Save(key []byte) error
}

const (
	keyService = "io.clipare.history"
	keyAccount = "master"
)

// defaultKeystore is installed by platform files.
var defaultKeystore Keystore

// DefaultKeystore returns the platform-protected keystore: Keychain on macOS,
// DPAPI-protected file on Windows. Without it history cannot be enabled.
func DefaultKeystore() (Keystore, error) {
	if defaultKeystore == nil {
		return nil, errors.New("history key storage is unavailable on this platform")
	}
	return defaultKeystore, nil
}
