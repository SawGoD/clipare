package security

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAuth(t *testing.T) {
	now := time.Unix(1000, 0)
	s := []byte("secret")
	body := []byte("hello")
	for _, offset := range []int64{-61, -60, 0, 60, 61} {
		ts := strconv.FormatInt(1000+offset, 10)
		sig := Sign(s, ts, body)
		if Verify(s, ts, sig, body, now) != (offset >= -60 && offset <= 60) {
			t.Fatal(offset)
		}
		if Verify(s, ts, sig, []byte("changed"), now) {
			t.Fatal("tamper")
		}
	}
	for _, ts := range []string{"+1000", "01000", "9223372036854775807", "-9223372036854775808", "bad"} {
		if Verify(s, ts, Sign(s, ts, body), body, now) {
			t.Fatal(ts)
		}
	}
	if Verify(s, "1000", strings.Repeat("0", 64), body, now) {
		t.Fatal("invalid signature")
	}
}
