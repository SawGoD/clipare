// Package pairing implements a committed X25519 handshake. A six-digit SAS is
// a human verification aid, never a password, transport key or lookup token.
package pairing

import (
	"clipare/internal/config"
	"clipare/internal/identity"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalid = errors.New("Не удалось проверить подключение. Попробуйте ещё раз")
var ErrExpired = errors.New("Pairing истёк, попробуйте ещё раз")
var ErrApproval = errors.New("Подключение ещё не подтверждено")
var ErrGroupMerge = errors.New("Объединение двух существующих групп пока не поддерживается")

type Hello struct {
	ExistingGroup string `json:"existing_group,omitempty"`
	Expires       int64  `json:"expires"`
	ID            string `json:"device_id"`
	Name          string `json:"device_name"`
	PublicKey     string `json:"public_key"`
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	Ephemeral     string `json:"ephemeral"`
	Nonce         string `json:"nonce"`
}
type Handshake struct {
	Hello     Hello
	ephemeral *ecdh.PrivateKey
	identity  identity.Provider
}

func (Handshake) String() string   { return "[redacted handshake]" }
func (Handshake) GoString() string { return "[redacted handshake]" }
func NewHandshake(c config.Config) (*Handshake, error) {
	k, e := ecdh.X25519().GenerateKey(rand.Reader)
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, 32)
	if _, e = rand.Read(nonce); e != nil {
		return nil, e
	}
	existing := ""
	if len(c.Peers) > 0 {
		existing = c.Group.ID
	}
	return &Handshake{Hello{ExistingGroup: existing, Expires: time.Now().Add(120 * time.Second).Unix(), ID: c.Device.ID, Name: c.Device.Name, PublicKey: c.Identity.PublicKey, IP: c.Listen.Address, Port: c.Listen.Port, Ephemeral: base64.StdEncoding.EncodeToString(k.PublicKey().Bytes()), Nonce: base64.StdEncoding.EncodeToString(nonce)}, k, c.Identity}, nil
}
func Commitment(h Hello) string {
	b, _ := json.Marshal(h)
	s := sha256.Sum256(append([]byte("clipare/pair/commit/v1\n"), b...))
	return hex.EncodeToString(s[:])
}

type transcript struct {
	Session   string `json:"session"`
	Expires   int64  `json:"expires"`
	Group     string `json:"group"`
	Initiator Hello  `json:"initiator"`
	Responder Hello  `json:"responder"`
}

func (h *Handshake) Keys(remote Hello, session, group string, expires int64, initiator bool) (key []byte, sas string, err error) {
	if remote.ID == "" || remote.ID == h.Hello.ID || len(remote.ID) > 128 || len(remote.Name) > 256 || strings.ContainsAny(remote.Name, "\r\n\x00") || len(session) != 64 || len(group) > 128 || group == "" || remote.Port < 1 || remote.Port > 65535 {
		return nil, "", ErrInvalid
	}
	n, e := base64.StdEncoding.DecodeString(remote.Nonce)
	if e != nil || len(n) != 32 {
		return nil, "", ErrInvalid
	}
	static, e := h.identity.Private()
	if e != nil {
		return nil, "", ErrInvalid
	}
	rs, e := identity.Public(remote.PublicKey)
	if e != nil {
		return nil, "", ErrInvalid
	}
	re, e := identity.Public(remote.Ephemeral)
	if e != nil {
		return nil, "", ErrInvalid
	}
	ss, e := static.ECDH(rs)
	if e != nil {
		return nil, "", ErrInvalid
	}
	ee, e := h.ephemeral.ECDH(re)
	if e != nil {
		return nil, "", ErrInvalid
	}
	se, e := static.ECDH(re)
	if e != nil {
		return nil, "", ErrInvalid
	}
	es, e := h.ephemeral.ECDH(rs)
	if e != nil {
		return nil, "", ErrInvalid
	}
	a, b := h.Hello, remote
	if !initiator {
		a, b = remote, h.Hello
		se, es = es, se
	}
	t, _ := json.Marshal(transcript{session, expires, group, a, b})
	salt := sha256.Sum256(t)
	material := append(append(append(ss, ee...), se...), es...)
	key = identity.Derive(material, salt[:], "pair/session")
	s := identity.Derive(key, salt[:], "pair/sas")
	code := binary.BigEndian.Uint32(s[:4]) % 1000000
	sas = fmt.Sprintf("%03d %03d", code/1000, code%1000)
	return key, sas, nil
}
func proof(key []byte, session, action string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte("clipare/pair/v1/" + session + "/" + action))
	return hex.EncodeToString(h.Sum(nil))
}
func verifyProof(key []byte, session, action, got string) bool {
	b, e := hex.DecodeString(got)
	if e != nil {
		return false
	}
	want, _ := hex.DecodeString(proof(key, session, action))
	return hmac.Equal(b, want)
}
