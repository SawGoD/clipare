package peers

import (
	"clipare/internal/config"
	"testing"
)

func TestThreeMemberMeshAndRemoval(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	c, _ := config.Default()
	a.Listen.Address = "100.64.0.1"
	b.Listen.Address = "100.64.0.2"
	c.Listen.Address = "100.64.0.3"
	b.Group = a.Group
	c.Group = a.Group
	m := Membership{Group: a.Group.ID, Members: []Member{Local(a), Local(b), Local(c)}}
	for _, original := range []config.Config{a, b, c} {
		got, err := Merge(original, m)
		if err != nil || len(got.Peers) != 2 {
			t.Fatal("mesh failed", err)
		}
	}
	a, err := Merge(a, m)
	if err != nil {
		t.Fatal(err)
	}
	a.Removed = []string{b.Device.ID}
	a, err = Merge(a, m)
	if err != nil || len(a.Peers) != 1 {
		t.Fatal("removed peer revived")
	}
	m.Members[2].PublicKey = b.Identity.PublicKey
	if _, err = Merge(a, m); err == nil {
		t.Fatal("key replacement accepted")
	}
}
