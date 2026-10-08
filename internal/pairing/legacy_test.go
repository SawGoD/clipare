package pairing

import (
	"bytes"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/identity"
	"clipare/internal/peers"
	"clipare/internal/security"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestAuthenticatedLegacyUpgrade(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	a.Listen.Address = "100.64.0.1"
	b.Listen.Address = "100.64.0.2"
	a, _ = config.AddConnection(a, config.ConnectionCode(b))
	b, _ = config.AddConnection(b, config.ConnectionCode(a))
	if a.Group.ID != b.Group.ID {
		t.Fatal("legacy groups diverged")
	}
	s := NewService(filepath.Join(t.TempDir(), "b.yaml"), b)
	// Disable throttling in this authentication test: Windows timestamps may
	// remain equal across consecutive requests. Rate limiting is tested separately.
	s.limiter = &discovery.Limiter{}
	nonce, _ := NewID()
	v := upgrade{Group: a.Group.ID, Member: peers.Local(a), Nonce: nonce}
	body, _ := json.Marshal(v)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	key := []byte(a.Security.Secret)
	call := func(signed bool, want int) []byte {
		r := httptest.NewRequest("POST", "/api/v1/group/upgrade", bytes.NewReader(body))
		r.RemoteAddr = a.Listen.Address + ":1234"
		r.Header.Set(security.SourceHeader, a.Device.ID)
		r.Header.Set(security.TimestampHeader, ts)
		if signed {
			r.Header.Set(security.SignatureHeader, security.Sign(key, ts, security.AuthenticatedData(r.Method, r.URL.Path, a.Device.ID, body)))
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("status %d want %d", w.Code, want)
		}
		return w.Body.Bytes()
	}
	call(false, 401)
	var out upgraded
	if json.Unmarshal(call(true, 200), &out) != nil {
		t.Fatal("response")
	}
	signed, _ := json.Marshal(out.Upgrade)
	if !security.Verify(key, ts, out.Signature, security.AuthenticatedData("RESPONSE", "/api/v1/group/upgrade", b.Device.ID, signed), time.Now()) {
		t.Fatal("unsigned public identity")
	}
	p := s.Config().Peers[0]
	if p.Legacy || p.PublicKey != a.Identity.PublicKey || p.LegacySecret != a.Security.Secret {
		t.Fatal("upgrade lost credentials")
	}
	call(true, 200) // retry after lost response cannot change the pinned key
	a.Listen.Address = "100.64.0.9"
	v.Member.IP = a.Listen.Address
	body, _ = json.Marshal(v)
	call(true, 200)
	if s.Config().Peers[0].LastKnownIP != "100.64.0.1" {
		t.Fatal("legacy retry redirected pinned metadata")
	}
	c, _ := config.Default()
	v.Member.PublicKey = c.Identity.PublicKey
	body, _ = json.Marshal(v)
	call(true, 400)
	metadata, _ := json.Marshal(peers.Export(s.Config()))
	pairKey, _ := identity.PairKey(a.Identity, b.Identity.PublicKey, b.Group.ID, a.Device.ID, b.Device.ID)
	r := httptest.NewRequest("POST", "/api/v1/group/membership", bytes.NewReader(metadata))
	r.RemoteAddr = a.Listen.Address + ":1234"
	r.Header.Set(security.SourceHeader, a.Device.ID)
	r.Header.Set(security.TimestampHeader, ts)
	r.Header.Set(security.SignatureHeader, security.Sign(pairKey, ts, security.AuthenticatedData(r.Method, r.URL.Path, a.Device.ID, metadata)))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 || s.Config().Peers[0].LegacySecret != "" {
		t.Fatal("legacy credential not retired after pairwise confirmation")
	}
	call(true, 401)
}
