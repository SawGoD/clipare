package peers

import (
	"clipare/internal/config"
	"testing"
)

func TestApprovedRejoinSurvivesStaleMembership(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	c, _ := config.Default()
	a.Listen.Address, b.Listen.Address, c.Listen.Address = "100.64.0.1", "100.64.0.2", "100.64.0.3"
	b.Group, c.Group = a.Group, a.Group
	m := Membership{Group: a.Group.ID, Members: []Member{Local(a), Local(b), Local(c)}}
	var err error
	a, err = Merge(a, m)
	if err != nil {
		t.Fatal(err)
	}
	c, err = Merge(c, m)
	if err != nil {
		t.Fatal(err)
	}
	old := Export(a)
	a = Remove(a, b.Device.ID)
	a, err = Merge(a, Export(a))
	if err != nil {
		t.Fatal(err)
	}
	deleted := Export(a)
	c, err = Merge(c, deleted)
	if err != nil || len(c.Peers) != 1 {
		t.Fatal("removal failed", err)
	}
	c, err = Merge(c, old)
	if err != nil || len(c.Peers) != 1 {
		t.Fatal("old membership revived device", err)
	}
	a, err = Reinstate(a, Membership{}, b.Device.ID)
	if err != nil {
		t.Fatal(err)
	}
	m.Versions = CloneVersions(a.MembershipVersions)
	a, err = Merge(a, m)
	if err != nil {
		t.Fatal(err)
	}
	restored := Export(a)
	c, err = Merge(c, restored)
	if err != nil || len(c.Peers) != 2 {
		t.Fatal("rejoin did not reach third device", err)
	}
	c, err = Merge(c, deleted)
	if err != nil || len(c.Peers) != 2 {
		t.Fatal("stale removal undid rejoin", err)
	}
	a = Remove(a, b.Device.ID)
	c, err = Merge(c, Export(a))
	if err != nil || len(c.Peers) != 1 {
		t.Fatal("second removal failed", err)
	}
	c, err = Merge(c, restored)
	if err != nil || len(c.Peers) != 1 {
		t.Fatal("stale rejoin undid second removal", err)
	}
}

func TestRevisionValidationAndOwnership(t *testing.T) {
	a, _ := config.Default()
	a.Listen.Address = "100.64.0.1"
	for _, revisions := range []map[string]uint64{{"b": 0}, {"b": 1 << 63}, {"": 1}, {"b": 2}} {
		v := Export(a)
		v.Versions = revisions
		if _, err := Merge(a, v); err == nil {
			t.Fatal("invalid or metadata-free restoration accepted", revisions)
		}
	}
	a.Removed = []string{"b"}
	a.MembershipVersions = map[string]uint64{"b": 1}
	restored, err := Reinstate(a, Membership{}, "b")
	if err != nil {
		t.Fatal(err)
	}
	if a.MembershipVersions["b"] != 1 || len(a.Removed) != 1 || restored.MembershipVersions["b"] != 2 {
		t.Fatal("reinstatement mutated original")
	}
	copy := Export(a)
	copy.Versions["b"] = 100
	if a.MembershipVersions["b"] != 1 {
		t.Fatal("export aliases mutable revisions")
	}
}
