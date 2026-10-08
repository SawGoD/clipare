package pairing

import (
	"clipare/internal/config"
	"clipare/internal/peers"
	"path/filepath"
	"testing"
)

func TestSettingsMergePreservesNewMemberAndRemoval(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	c, _ := config.Default()
	a.Listen.Address = "100.64.0.1"
	b.Listen.Address = "100.64.0.2"
	c.Listen.Address = "100.64.0.3"
	b.Group = a.Group
	c.Group = a.Group
	a, _ = peers.Merge(a, peers.Membership{Group: a.Group.ID, Members: []peers.Member{peers.Local(a), peers.Local(b)}})
	s := NewService(filepath.Join(t.TempDir(), "a.yaml"), a)
	newMembership := peers.Export(a)
	newMembership.Members = append(newMembership.Members, peers.Local(c))
	if err := s.Adopt(newMembership, b.Device.ID); err != nil {
		t.Fatal(err)
	}
	draft := a
	draft.Device.Name = "Renamed"
	saved, err := s.SaveSettings(draft)
	if err != nil || len(saved.Peers) != 2 || saved.Device.Name != "Renamed" {
		t.Fatal("concurrent member lost", err)
	}
	saved.Removed = []string{b.Device.ID}
	saved.Peers = saved.Peers[1:]
	saved, err = s.SaveSettings(saved)
	if err != nil || len(saved.Peers) != 1 || saved.Peers[0].ID != c.Device.ID {
		t.Fatal("removal undone", err)
	}
}
