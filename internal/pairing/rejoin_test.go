package pairing

import (
	"bytes"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/peers"
	"clipare/internal/security"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestRemovedDeviceRequiresFreshApprovedPairing(t *testing.T) {
	for _, allow := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject", true: "approve"}[allow], func(t *testing.T) {
			a, _ := config.Default()
			b, _ := config.Default()
			a.Listen.Address, b.Listen.Address = "100.64.0.1", "100.64.0.2"
			b.Group = a.Group
			m := peers.Membership{Group: a.Group.ID, Members: []peers.Member{peers.Local(a), peers.Local(b)}}
			var err error
			a, err = peers.Merge(a, m)
			if err != nil {
				t.Fatal(err)
			}
			b, err = peers.Merge(b, m)
			if err != nil {
				t.Fatal(err)
			}
			a = peers.Remove(a, b.Device.ID)
			a, err = peers.Merge(a, peers.Export(a))
			if err != nil {
				t.Fatal(err)
			}
			b = peers.Remove(b, a.Device.ID)
			b, err = peers.Merge(b, peers.Export(b))
			if err != nil {
				t.Fatal(err)
			}
			sa := NewService(filepath.Join(t.TempDir(), "a.yaml"), a)
			sb := NewService(filepath.Join(t.TempDir(), "b.yaml"), b)
			sb.limiter.Interval = 0
			sa.client = &http.Client{Transport: directTransport{sb, a.Listen.Address}}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			decision := make(chan error, 1)
			code := make(chan string, 1)
			go func() {
				tick := time.NewTicker(time.Millisecond)
				defer tick.Stop()
				for {
					select {
					case <-ctx.Done():
						decision <- ctx.Err()
						return
					case <-tick.C:
						if snap, ok := sb.Snapshot(); ok && snap.State == Pending {
							var shown string
							select {
							case shown = <-code:
							case <-ctx.Done():
								decision <- ctx.Err()
								return
							}
							if snap.SAS != shown {
								decision <- errors.New("SAS mismatch")
								return
							}
							decision <- sb.Decide(snap.Session, allow)
							return
						}
					}
				}
			}()
			err = sa.Connect(ctx, discovery.Device{Peer: discovery.Peer{IP: b.Listen.Address}, Info: discovery.Info{DeviceID: b.Device.ID, PublicKey: b.Identity.PublicKey}}, func(s string) { code <- s })
			if e := <-decision; e != nil {
				t.Fatal(e)
			}
			if !allow {
				if err == nil || len(sa.Config().Peers) != 0 || sa.Config().MembershipVersions[b.Device.ID] != 1 {
					t.Fatal("rejection changed trust", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(sa.Config().Peers) != 1 || len(sb.Config().Peers) != 1 || len(sa.Config().Removed) != 0 || sa.Config().MembershipVersions[b.Device.ID] != 2 || sb.Config().MembershipVersions[a.Device.ID] != 2 {
				t.Fatal("approved rejoin not persisted")
			}
			if sa.Config().Identity != a.Identity || sb.Config().Identity != b.Identity {
				t.Fatal("identity reset during rejoin")
			}
			loaded, e := config.Load(sa.path)
			if e != nil || loaded.MembershipVersions[b.Device.ID] != 2 {
				t.Fatal("revision not persisted", e)
			}
		})
	}
}

func TestRemovedDeviceCannotReinstateItselfWithMembership(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	a.Listen.Address, b.Listen.Address = "100.64.0.1", "100.64.0.2"
	b.Group = a.Group
	m := peers.Membership{Group: a.Group.ID, Members: []peers.Member{peers.Local(a), peers.Local(b)}}
	a, err := peers.Merge(a, m)
	if err != nil {
		t.Fatal(err)
	}
	key, err := security.NewPeerKeys(a).KeyForPeer(b.Device.ID)
	if err != nil {
		t.Fatal(err)
	}
	a = peers.Remove(a, b.Device.ID)
	a, err = peers.Merge(a, peers.Export(a))
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(filepath.Join(t.TempDir(), "a.yaml"), a)
	m.Versions = map[string]uint64{b.Device.ID: 2}
	body, _ := json.Marshal(m)
	r := httptest.NewRequest("POST", "/api/v1/group/membership", bytes.NewReader(body))
	r.RemoteAddr = b.Listen.Address + ":1234"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r.Header.Set(security.SourceHeader, b.Device.ID)
	r.Header.Set(security.TimestampHeader, ts)
	r.Header.Set(security.SignatureHeader, security.Sign(key, ts, security.AuthenticatedData(r.Method, r.URL.Path, b.Device.ID, body)))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || len(s.Config().Peers) != 0 {
		t.Fatal("removed peer reinstated itself", w.Code)
	}
}
