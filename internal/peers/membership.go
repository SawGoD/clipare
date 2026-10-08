package peers

import (
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/identity"
	"errors"
	"sort"
	"strings"
)

type Member struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	FQDN      string `json:"fqdn,omitempty"`
	Port      int    `json:"port"`
	PublicKey string `json:"public_key"`
}
type Membership struct {
	Group   string   `json:"group"`
	Members []Member `json:"members"`
	Removed []string `json:"removed,omitempty"`
}

func Local(c config.Config) Member {
	return Member{ID: c.Device.ID, Name: c.Device.Name, IP: c.Listen.Address, Port: c.Listen.Port, PublicKey: c.Identity.PublicKey}
}
func Export(c config.Config) Membership {
	v := Membership{Group: c.Group.ID, Members: []Member{Local(c)}, Removed: append([]string(nil), c.Removed...)}
	for _, p := range c.Peers {
		if p.Legacy {
			continue
		}
		m := Member{ID: p.ID, Name: p.Name, IP: p.LastKnownIP, Port: p.Port, PublicKey: p.PublicKey}
		if discovery.NetBirdAddress(p.Address) {
			m.IP = p.Address
		} else {
			m.FQDN = p.Address
		}
		v.Members = append(v.Members, m)
	}
	return v
}
func (m Member) Validate() error {
	if m.ID == "" || len(m.ID) > 128 || len(m.Name) > 256 || strings.ContainsAny(m.Name, "\n\r\x00") || strings.ContainsAny(m.ID, "\n\r/") || !discovery.NetBirdAddress(m.IP) || m.Port < 1 || m.Port > 65535 {
		return errors.New("invalid group member")
	}
	if m.FQDN != "" && (len(m.FQDN) > 253 || strings.ContainsAny(m.FQDN, "/\\?#@ :\t\r\n")) {
		return errors.New("invalid member hostname")
	}
	_, err := identity.Public(m.PublicKey)
	return err
}
func Merge(c config.Config, v Membership) (config.Config, error) {
	if v.Group != c.Group.ID || len(v.Members) > 65 || len(v.Removed) > 256 {
		return c, errors.New("invalid group membership")
	}
	seen := map[string]bool{}
	for _, m := range v.Members {
		if m.Validate() != nil || seen[m.ID] {
			return c, errors.New("invalid group membership")
		}
		seen[m.ID] = true
		if m.ID == c.Device.ID && m.PublicKey != c.Identity.PublicKey {
			return c, errors.New("identity substitution")
		}
	}
	revoked := map[string]bool{}
	for _, id := range c.Removed {
		revoked[id] = true
	}
	for _, id := range v.Removed {
		if id == c.Device.ID || id == "" || len(id) > 128 {
			return c, errors.New("invalid removal")
		}
		revoked[id] = true
	}
	if len(revoked) > 256 {
		return c, errors.New("removal list full")
	}
	c.Peers = append([]config.Peer(nil), c.Peers...)
	c.Removed = nil
	for id := range revoked {
		c.Removed = append(c.Removed, id)
	}
	sort.Strings(c.Removed)
	for _, m := range v.Members {
		if m.ID == c.Device.ID || revoked[m.ID] {
			continue
		}
		found := false
		for j, p := range c.Peers {
			if p.ID != m.ID {
				continue
			}
			found = true
			if p.Legacy || p.PublicKey != m.PublicKey {
				return c, errors.New("peer identity changed")
			}
			c.Peers[j].Name = m.Name
			c.Peers[j].LastKnownIP = m.IP
			c.Peers[j].Address = m.IP
			if !discovery.NetBirdAddress(p.Address) {
				c.Peers[j].Address = p.Address
			}
			if m.FQDN != "" {
				c.Peers[j].Address = m.FQDN
			}
			c.Peers[j].Port = m.Port
			break
		}
		if !found {
			address := m.IP
			if m.FQDN != "" {
				address = m.FQDN
			}
			c.Peers = append(c.Peers, config.Peer{ID: m.ID, Name: m.Name, Address: address, LastKnownIP: m.IP, Port: m.Port, PublicKey: m.PublicKey})
		}
	}
	kept := c.Peers[:0]
	for _, p := range c.Peers {
		if !revoked[p.ID] {
			kept = append(kept, p)
		}
	}
	c.Peers = kept
	return c, c.Validate()
}
