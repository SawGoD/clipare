package security

import (
	"bytes"
	"clipare/internal/config"
	"testing"
)

func TestPeerCredentials(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	c, _ := config.Default()
	b.Group = a.Group
	c.Group = a.Group
	a.Peers = []config.Peer{{ID: b.Device.ID, PublicKey: b.Identity.PublicKey}, {ID: c.Device.ID, PublicKey: c.Identity.PublicKey}}
	b.Peers = []config.Peer{{ID: a.Device.ID, PublicKey: a.Identity.PublicKey}}
	ab, e := NewPeerKeys(a).KeyForPeer(b.Device.ID)
	if e != nil {
		t.Fatal(e)
	}
	ba, _ := NewPeerKeys(b).KeyForPeer(a.Device.ID)
	ac, _ := NewPeerKeys(a).KeyForPeer(c.Device.ID)
	if !bytes.Equal(ab, ba) || bytes.Equal(ab, ac) {
		t.Fatal("pairwise isolation")
	}
	a.Peers = a.Peers[1:]
	if _, e = NewPeerKeys(a).KeyForPeer(b.Device.ID); e == nil {
		t.Fatal("removed peer authenticated")
	}
	legacy := config.Peer{ID: "old", Legacy: true, LegacySecret: a.Security.Secret}
	a.Peers = append(a.Peers, legacy)
	k, e := NewPeerKeys(a).KeyForPeer("old")
	if e != nil || string(k) != a.Security.Secret {
		t.Fatal("legacy key lost")
	}
}
