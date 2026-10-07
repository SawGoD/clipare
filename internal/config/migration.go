package config

import (
	"clipare/internal/identity"
	"errors"
	"os"
)

// Upgrade is pure with respect to storage: validation finishes before any write.
func Upgrade(c Config) (Config, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if c.SchemaVersion == 2 {
		return c, nil
	}
	i, err := identity.Generate()
	if err != nil {
		return c, err
	}
	g, err := NewSecret()
	if err != nil {
		return c, err
	}
	c.SchemaVersion = 2
	c.Identity = i
	c.Group.ID = g
	c.Peers = append([]Peer(nil), c.Peers...)
	for j := range c.Peers {
		c.Peers[j].Legacy = true
		c.Peers[j].LegacySecret = c.Security.Secret
	}
	return c, c.Validate()
}

// LoadMigrated retains an exact owner-only backup before atomic replacement.
// Existing backups are never overwritten. Failure leaves the original usable.
func LoadMigrated(path string) (Config, error) {
	c, err := Load(path)
	if err != nil || c.SchemaVersion == 2 {
		return c, err
	}
	next, err := Upgrade(c)
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return c, errors.New("cannot back up configuration")
	}
	f, err := os.OpenFile(path+".v1.backup", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return c, errors.New("migration backup already exists or cannot be created")
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err == nil {
		err = ce
	}
	if err != nil {
		return c, errors.New("cannot back up configuration")
	}
	if err = Save(path, next); err != nil {
		return c, err
	}
	return next, nil
}
