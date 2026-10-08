package pairing

import (
	"bytes"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/identity"
	"clipare/internal/peers"
	"clipare/internal/security"
	"context"
	"crypto/hmac"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type directTransport struct {
	handler http.Handler
	source  string
}

func (t directTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.RemoteAddr = t.source + ":1234"
	w := httptest.NewRecorder()
	t.handler.ServeHTTP(w, r)
	return w.Result(), nil
}
func TestConnectAndThreeDeviceMetadata(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	c, _ := config.Default()
	a.Listen.Address = "100.64.0.1"
	b.Listen.Address = "100.64.0.2"
	c.Listen.Address = "100.64.0.3"
	c.Group = b.Group
	b, err := peers.Merge(b, peers.Membership{Group: b.Group.ID, Members: []peers.Member{peers.Local(b), peers.Local(c)}})
	if err != nil {
		t.Fatal(err)
	}
	sa := NewService(filepath.Join(t.TempDir(), "a.yaml"), a)
	sb := NewService(filepath.Join(t.TempDir(), "b.yaml"), b)
	sb.limiter.Interval = 0
	sa.client = &http.Client{Transport: directTransport{sb, a.Listen.Address}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	approved := make(chan string, 1)
	go func() {
		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if snap, ok := sb.Snapshot(); ok && snap.State == Pending {
					approved <- snap.SAS
					sb.Decide(snap.Session, true)
					return
				}
			}
		}
	}()
	var sas string
	err = sa.Connect(ctx, discovery.Device{Peer: discovery.Peer{IP: b.Listen.Address}, Info: discovery.Info{DeviceID: b.Device.ID, PublicKey: b.Identity.PublicKey}}, func(code string) { sas = code })
	if err != nil {
		t.Fatal(err)
	}
	if sas != <-approved {
		t.Fatal("SAS mismatch")
	}
	if len(sa.Config().Peers) != 2 || len(sb.Config().Peers) != 2 {
		t.Fatal("pair metadata mesh")
	}
	c, err = peers.Merge(c, peers.Membership{Group: c.Group.ID, Members: []peers.Member{peers.Local(b), peers.Local(c)}})
	if err != nil {
		t.Fatal(err)
	}
	sc := NewService(filepath.Join(t.TempDir(), "c.yaml"), c)
	body, _ := json.Marshal(peers.Export(sb.Config()))
	key, _ := security.NewPeerKeys(sb.Config()).KeyForPeer(c.Device.ID)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r := httptest.NewRequest("POST", "/api/v1/group/membership", bytes.NewReader(body))
	r.RemoteAddr = b.Listen.Address + ":1234"
	r.Header.Set(security.SourceHeader, b.Device.ID)
	r.Header.Set(security.TimestampHeader, ts)
	r.Header.Set(security.SignatureHeader, security.Sign(key, ts, security.AuthenticatedData(r.Method, r.URL.Path, b.Device.ID, body)))
	w := httptest.NewRecorder()
	sc.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("third member update status %d", w.Code)
	}
	c = sc.Config()
	if err != nil || len(c.Peers) != 2 {
		t.Fatal("third member metadata", err)
	}
	a = sa.Config()
	ab, _ := identity.PairKey(a.Identity, b.Identity.PublicKey, a.Group.ID, a.Device.ID, b.Device.ID)
	ba, _ := identity.PairKey(b.Identity, a.Identity.PublicKey, b.Group.ID, b.Device.ID, a.Device.ID)
	ac, _ := identity.PairKey(a.Identity, c.Identity.PublicKey, a.Group.ID, a.Device.ID, c.Device.ID)
	if !hmac.Equal(ab, ba) || bytes.Equal(ab, ac) {
		t.Fatal("mesh pair key mismatch")
	}
}
