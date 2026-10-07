// Package identity owns persistent X25519 device identities. Storage is kept
// outside this package so a keychain-backed provider can replace config storage.
package identity

import (
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sort"
)

type Identity struct {
	PrivateKey string `yaml:"private_key" json:"-"`
	PublicKey  string `yaml:"public_key" json:"public_key"`
}

func (Identity) String() string   { return "[redacted identity]" }
func (Identity) GoString() string { return "[redacted identity]" }

type Provider interface {
	Private() (*ecdh.PrivateKey, error)
}

func Generate() (Identity, error) {
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return Identity{}, err
	}
	return Identity{base64.StdEncoding.EncodeToString(k.Bytes()), base64.StdEncoding.EncodeToString(k.PublicKey().Bytes())}, nil
}

func (i Identity) Private() (*ecdh.PrivateKey, error) {
	b, err := base64.StdEncoding.DecodeString(i.PrivateKey)
	if err != nil {
		return nil, errors.New("invalid device identity")
	}
	k, err := ecdh.X25519().NewPrivateKey(b)
	if err != nil || base64.StdEncoding.EncodeToString(k.PublicKey().Bytes()) != i.PublicKey {
		return nil, errors.New("invalid device identity")
	}
	return k, nil
}

func Public(encoded string) (*ecdh.PublicKey, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("invalid public key")
	}
	k, err := ecdh.X25519().NewPublicKey(b)
	if err != nil {
		return nil, errors.New("invalid public key")
	}
	// ECDH also rejects low-order/all-zero keys, which NewPublicKey alone accepts.
	probe, err := ecdh.X25519().NewPrivateKey(make([]byte, 32))
	if err != nil {
		return nil, err
	}
	if _, err = probe.ECDH(k); err != nil {
		return nil, errors.New("invalid public key")
	}
	return k, nil
}

// Derive is HKDF-SHA256 (RFC 5869), one output block, with domain separation.
func Derive(secret, salt []byte, purpose string) []byte {
	extract := hmac.New(sha256.New, salt)
	extract.Write(secret)
	expand := hmac.New(sha256.New, extract.Sum(nil))
	expand.Write([]byte("clipare/v2/" + purpose))
	expand.Write([]byte{1})
	return expand.Sum(nil)
}

// PairKey is symmetric but bound to the group and both device IDs. Membership
// distributes public keys only; no member learns another pair's secret.
func PairKey(local Provider, remote, group, localID, remoteID string) ([]byte, error) {
	k, err := local.Private()
	if err != nil {
		return nil, err
	}
	p, err := Public(remote)
	if err != nil {
		return nil, err
	}
	s, err := k.ECDH(p)
	if err != nil {
		return nil, err
	}
	ids := []string{localID, remoteID}
	sort.Strings(ids)
	return Derive(s, []byte(group), "clipboard/"+ids[0]+"/"+ids[1]), nil
}
