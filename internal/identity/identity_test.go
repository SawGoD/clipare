package identity

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestIdentityAndPairKey(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Generate()
	c, _ := Generate()
	ab, err := PairKey(a, b.PublicKey, "g", "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	ba, _ := PairKey(b, a.PublicKey, "g", "b", "a")
	ac, _ := PairKey(a, c.PublicKey, "g", "a", "c")
	if !bytes.Equal(ab, ba) || bytes.Equal(ab, ac) {
		t.Fatal("pair key isolation")
	}
	other, _ := PairKey(a, b.PublicKey, "other", "a", "b")
	if bytes.Equal(ab, other) {
		t.Fatal("group binding")
	}
	for _, bad := range []string{"bad", base64.StdEncoding.EncodeToString(make([]byte, 32))} {
		if _, err := Public(bad); err == nil {
			t.Fatal("accepted invalid public key")
		}
	}
	a.PublicKey = b.PublicKey
	if _, err := a.Private(); err == nil {
		t.Fatal("mismatched identity accepted")
	}
	if bytes.Contains([]byte(fmt.Sprintf("%+v %#v", a, a)), []byte(a.PrivateKey)) {
		t.Fatal("identity leaked")
	}
}
