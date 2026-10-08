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

func TestExistingMemberInitiatesToNewDevice(t *testing.T) {
	a, _ := config.Default()
	old, _ := config.Default()
	fresh, _ := config.Default()
	a.Listen.Address = "100.64.0.1"
	old.Listen.Address = "100.64.0.2"
	fresh.Listen.Address = "100.64.0.3"
	old.Group = a.Group
	a, _ = peers.Merge(a, peers.Membership{Group: a.Group.ID, Members: []peers.Member{peers.Local(a), peers.Local(old)}})
	sa := NewService(filepath.Join(t.TempDir(), "a.yaml"), a)
	sb := NewService(filepath.Join(t.TempDir(), "fresh.yaml"), fresh)
	sb.limiter.Interval = 0
	sa.client = &http.Client{Transport: directTransport{sb, a.Listen.Address}}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	remoteApproved := make(chan error, 1)
	go func() {
		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if v, ok := sb.Snapshot(); ok && v.State == Pending {
					remoteApproved <- sb.Decide(v.Session, true)
					return
				}
			}
		}
	}()
	verified := false
	err := sa.Connect(ctx, discovery.Device{Peer: discovery.Peer{IP: fresh.Listen.Address}, Info: discovery.Info{DeviceID: fresh.Device.ID, PublicKey: fresh.Identity.PublicKey}}, func(string) {}, func(ctx context.Context) error {
		select {
		case err := <-remoteApproved:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		if len(sb.Config().Peers) != 0 {
			t.Fatal("new device persisted before existing member verified SAS")
		}
		verified = true
		return nil
	})
	if err != nil || !verified {
		t.Fatal("existing initiated pairing", err)
	}
	if sb.Config().Group.ID != a.Group.ID || sa.Config().Group.ID != a.Group.ID || len(sa.Config().Peers) != 2 {
		t.Fatal("existing group lost")
	}
}
func TestRejectAndDifferentEstablishedGroups(t *testing.T) {
	for _, differentGroups := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject", true: "group_merge"}[differentGroups], func(t *testing.T) {
			a, _ := config.Default()
			b, _ := config.Default()
			a.Listen.Address = "100.64.0.1"
			b.Listen.Address = "100.64.0.2"
			if differentGroups {
				other, _ := config.Default()
				other.Listen.Address = "100.64.0.3"
				a, _ = peers.Merge(a, peers.Membership{Group: a.Group.ID, Members: []peers.Member{peers.Local(a), peers.Local(other)}})
				b, _ = peers.Merge(b, peers.Membership{Group: b.Group.ID, Members: []peers.Member{peers.Local(b), peers.Local(other)}})
			}
			sa := NewService(filepath.Join(t.TempDir(), "a.yaml"), a)
			sb := NewService(filepath.Join(t.TempDir(), "b.yaml"), b)
			sb.limiter.Interval = 0
			sa.client = &http.Client{Transport: directTransport{sb, a.Listen.Address}}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if !differentGroups {
				go func() {
					tick := time.NewTicker(time.Millisecond)
					defer tick.Stop()
					for {
						select {
						case <-ctx.Done():
							return
						case <-tick.C:
							if v, ok := sb.Snapshot(); ok && v.State == Pending {
								sb.Decide(v.Session, false)
								return
							}
						}
					}
				}()
			}
			err := sa.Connect(ctx, discovery.Device{Peer: discovery.Peer{IP: b.Listen.Address}, Info: discovery.Info{DeviceID: b.Device.ID, PublicKey: b.Identity.PublicKey}}, func(string) {}, func(context.Context) error { return nil })
			if err == nil {
				t.Fatal("unexpected pairing success")
			}
			if differentGroups && !errors.Is(err, ErrGroupMerge) {
				t.Fatal(err)
			}
			if len(sa.Config().Peers) != len(a.Peers) || len(sb.Config().Peers) != len(b.Peers) {
				t.Fatal("failed pairing changed trust")
			}
		})
	}
}
