package history

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

const (
	keySize     = 32
	nonceSize   = 12
	formatTag   = "clipare-history-v1"
	recordLimit = 1<<20 + 4096 // payload limit plus envelope overhead
)

// ErrKeyLost reports that stored records cannot be authenticated with the
// current key. There is deliberately no fallback key: the user must clear
// the history instead of reading ciphertext as plaintext.
var ErrKeyLost = errors.New("history encryption key is unavailable or was replaced")

type box struct {
	gcm cipher.AEAD
}

func newBox(key []byte) (*box, error) {
	if len(key) != keySize {
		return nil, errors.New("history key must contain 32 bytes")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	return &box{gcm: gcm}, nil
}

// seal produces nonce||ciphertext, binding the record ID as AAD so blobs
// cannot be swapped between rows.
func (b *box) seal(id string, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, nonceSize)
	if _, e := rand.Read(nonce); e != nil {
		return nil, e
	}
	out := make([]byte, 0, nonceSize+len(plaintext)+b.gcm.Overhead())
	out = append(out, nonce...)
	out = b.gcm.Seal(out, nonce, plaintext, aad(id))
	return out, nil
}

func (b *box) open(id string, blob []byte) ([]byte, error) {
	if len(blob) < nonceSize+b.gcm.Overhead() || len(blob) > recordLimit {
		return nil, ErrKeyLost
	}
	plaintext, e := b.gcm.Open(nil, blob[:nonceSize], blob[nonceSize:], aad(id))
	if e != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyLost, e)
	}
	return plaintext, nil
}

func aad(id string) []byte { return []byte(formatTag + "|" + id) }
