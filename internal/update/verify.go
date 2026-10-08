package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

var ErrVerify = errors.New("Не удалось проверить обновление")

// ManifestVerifier is independent of downloading/staging. TODO(release-signing):
// embed a dedicated public key and require SHA256SUMS.sig once CI is provisioned.
// SHA256SUMS alone provides integrity, not independent publisher authentication.
type ManifestVerifier interface {
	Verify(manifest, signature []byte) error
	RequiresSignature() bool
}
type IntegrityOnly struct{}

func (IntegrityOnly) Verify([]byte, []byte) error { return nil }
func (IntegrityOnly) RequiresSignature() bool     { return false }

type Ed25519Verifier struct{ PublicKey ed25519.PublicKey }

func (Ed25519Verifier) RequiresSignature() bool { return true }
func (v Ed25519Verifier) Verify(m, s []byte) error {
	if len(v.PublicKey) != ed25519.PublicKeySize || !ed25519.Verify(v.PublicKey, m, s) {
		return ErrVerify
	}
	return nil
}
func Checksums(b []byte) (map[string]string, error) {
	if len(b) == 0 || len(b) > 64<<10 {
		return nil, ErrVerify
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if len(line) < 67 || line[64:66] != "  " {
			return nil, ErrVerify
		}
		sum, name := line[:64], line[66:]
		decoded, e := hex.DecodeString(sum)
		if e != nil || len(decoded) != 32 || !safeRelative(name) || strings.Contains(name, "/") || out[name] != "" {
			return nil, ErrVerify
		}
		out[name] = strings.ToLower(sum)
	}
	return out, nil
}
func FileSHA256(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func VerifyFile(path, want string) error {
	got, e := FileSHA256(path)
	if e != nil || got != want {
		return ErrVerify
	}
	return nil
}
