package pairing

import (
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/peers"
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestPairingRepairsOneSidedRelationship(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	a.Listen.Address, b.Listen.Address = "100.64.0.1", "100.64.0.2"
	b, err := peers.Merge(b, peers.Membership{Group: b.Group.ID, Members: []peers.Member{peers.Local(b), peers.Local(a)}})
	if err != nil {
		t.Fatal(err)
	}
	sa := NewService(filepath.Join(t.TempDir(), "a.yaml"), a)
	sb := NewService(filepath.Join(t.TempDir(), "b.yaml"), b)
	sb.limiter.Interval = 0
	sa.client = &http.Client{Transport: directTransport{sb, a.Listen.Address}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	approved := make(chan error, 1)
	go func() {
		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if snap, ok := sb.Snapshot(); ok && snap.State == Pending {
					approved <- sb.Decide(snap.Session, true)
					return
				}
			}
		}
	}()
	err = sa.Connect(ctx, discovery.Device{Peer: discovery.Peer{IP: b.Listen.Address}, Info: discovery.Info{DeviceID: b.Device.ID, PublicKey: b.Identity.PublicKey}}, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if err := <-approved; err != nil {
		t.Fatal(err)
	}
	if len(sa.Config().Peers) != 1 || len(sb.Config().Peers) != 1 || sa.Config().Group.ID != b.Group.ID {
		t.Fatal("relationship was not repaired without duplicates")
	}
}

func TestRepairKeepsIdentityPinnedAndRequiresApprovalForRevokedPeer(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	a.Listen.Address, b.Listen.Address = "100.64.0.1", "100.64.0.2"
	b, _ = peers.Merge(b, peers.Membership{Group: b.Group.ID, Members: []peers.Member{peers.Local(b), peers.Local(a)}})
	remote := Hello{ID: a.Device.ID, Name: a.Device.Name, IP: a.Listen.Address, Port: a.Listen.Port, PublicKey: a.Identity.PublicKey}
	s := NewService(filepath.Join(t.TempDir(), "b.yaml"), b)
	other, _ := config.Default()
	remote.PublicKey = other.Identity.PublicKey
	if err := s.persistPair(remote, b.Group.ID); !errors.Is(err, ErrIdentityChanged) {
		t.Fatal("changed identity accepted")
	}
	remote.PublicKey = a.Identity.PublicKey
	b.Removed = []string{a.Device.ID}
	s = NewService(filepath.Join(t.TempDir(), "revoked.yaml"), b)
	if err := s.Adopt(peers.Membership{Group: b.Group.ID, Members: []peers.Member{peers.Local(a), peers.Local(b)}}, a.Device.ID); !errors.Is(err, ErrRemoved) {
		t.Fatal("pairing reported success without saving the revoked peer", err)
	}
}

func TestPairingRecoveryErrorsReachInitiator(t *testing.T) {
	for status, want := range map[int]error{422: ErrIdentityChanged, 423: ErrRemoved} {
		client := &http.Client{Transport: directTransport{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}), "100.64.0.1"}}
		if err := post(context.Background(), client, "http://100.64.0.2", "/api/v1/pair/confirm", Confirmation{}, nil); !errors.Is(err, want) {
			t.Fatalf("status %d lost actionable error: %v", status, err)
		}
	}
}
