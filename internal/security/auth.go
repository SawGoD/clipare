package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

const TimestampHeader = "X-Clipare-Timestamp"
const SignatureHeader = "X-Clipare-Signature"

// SecretProvider allows replacing config storage with Keychain/Credential Manager.
type SecretProvider interface{ Secret() []byte }
type StaticSecret string

func (s StaticSecret) Secret() []byte { return []byte(s) }
func (StaticSecret) String() string   { return "[redacted]" }
func (StaticSecret) GoString() string { return "[redacted]" }
func Sign(secret []byte, timestamp string, body []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(timestamp))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
func Verify(secret []byte, timestamp, signature string, body []byte, now time.Time) bool {
	ts, e := strconv.ParseInt(timestamp, 10, 64)
	if e != nil || strconv.FormatInt(ts, 10) != timestamp {
		return false
	}
	n := now.Unix()
	if ts < n-60 || ts > n+60 {
		return false
	}
	got, e := hex.DecodeString(signature)
	if e != nil {
		return false
	}
	want, _ := hex.DecodeString(Sign(secret, timestamp, body))
	return hmac.Equal(got, want)
}
