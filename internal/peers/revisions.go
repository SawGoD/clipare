package peers

import (
	"clipare/internal/config"
	"errors"
)

func CloneVersions(v map[string]uint64) map[string]uint64 {
	if len(v) == 0 {
		return nil
	}
	copy := make(map[string]uint64, len(v))
	for id, revision := range v {
		copy[id] = revision
	}
	return copy
}

func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Legacy tombstones have revision 1. Ties favor removal. A freshly approved
// pairing advances the revision, so old membership packets cannot undo it.
func revision(versions map[string]uint64, removed []string, id string) uint64 {
	v := versions[id]
	if v == 0 && contains(removed, id) {
		return 1
	}
	return v
}

// Reinstate is only called from a successfully verified, user-approved pairing.
// Ordinary membership reconciliation must never call it.
func Reinstate(c config.Config, observed Membership, id string) (config.Config, error) {
	if !contains(c.Removed, id) {
		return c, nil
	}
	v := max(revision(c.MembershipVersions, c.Removed, id), revision(observed.Versions, observed.Removed, id))
	if v >= (1<<63)-1 {
		return c, errors.New("membership revision exhausted")
	}
	c.MembershipVersions = CloneVersions(c.MembershipVersions)
	if c.MembershipVersions == nil {
		c.MembershipVersions = map[string]uint64{}
	}
	c.MembershipVersions[id] = v + 1
	removed := make([]string, 0, len(c.Removed))
	for _, old := range c.Removed {
		if old != id {
			removed = append(removed, old)
		}
	}
	c.Removed = removed
	return c, nil
}

func Remove(c config.Config, id string) config.Config {
	c.MembershipVersions = CloneVersions(c.MembershipVersions)
	if c.MembershipVersions == nil {
		c.MembershipVersions = map[string]uint64{}
	}
	c.MembershipVersions[id] = revision(c.MembershipVersions, c.Removed, id) + 1
	c.Removed = append(append([]string(nil), c.Removed...), id)
	return c
}

func mergeRevisions(c config.Config, v Membership, members map[string]bool) (map[string]bool, map[string]uint64, error) {
	if len(v.Versions) > 256 {
		return nil, nil, errors.New("membership version limit exceeded")
	}
	ids := map[string]bool{}
	for _, id := range c.Removed {
		ids[id] = true
	}
	for _, id := range v.Removed {
		ids[id] = true
	}
	for id := range c.MembershipVersions {
		ids[id] = true
	}
	for id, r := range v.Versions {
		if r == 0 || r >= 1<<63 {
			return nil, nil, errors.New("invalid membership revision")
		}
		ids[id] = true
	}
	if len(ids) > 256 {
		return nil, nil, errors.New("membership version limit exceeded")
	}
	revoked := map[string]bool{}
	var versions map[string]uint64
	for id := range ids {
		if id == "" || len(id) > 128 {
			return nil, nil, errors.New("invalid removal")
		}
		local, remote := revision(c.MembershipVersions, c.Removed, id), revision(v.Versions, v.Removed, id)
		removed := contains(c.Removed, id)
		if remote > local {
			removed = contains(v.Removed, id)
			if !removed && !members[id] {
				return nil, nil, errors.New("restoration requires member metadata")
			}
		} else if remote == local {
			removed = removed || contains(v.Removed, id)
		}
		if removed {
			if id == c.Device.ID {
				return nil, nil, errors.New("invalid self removal")
			}
			revoked[id] = true
		}
		latest := max(local, remote)
		if latest > 0 {
			if versions == nil {
				versions = map[string]uint64{}
			}
			versions[id] = latest
		}
	}
	return revoked, versions, nil
}
