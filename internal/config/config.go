package config

import (
	"bytes"
	"clipare/internal/identity"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type Device struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}
type Peer struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	Address      string `yaml:"address"`
	Port         int    `yaml:"port"`
	PublicKey    string `yaml:"public_key,omitempty"`
	Legacy       bool   `yaml:"legacy,omitempty"`
	LegacySecret string `yaml:"legacy_secret,omitempty" json:"-"`
	LastKnownIP  string `yaml:"last_known_ip,omitempty"`
}

func (p Peer) String() string   { return "peer:" + p.ID }
func (p Peer) GoString() string { return p.String() }

func (p Peer) URL() string { return "http://" + net.JoinHostPort(p.Address, strconv.Itoa(p.Port)) }

type Security struct {
	Secret string `yaml:"secret"`
}

// String prevents accidental disclosure in formatted diagnostics.
func (Security) String() string   { return "[redacted]" }
func (Security) GoString() string { return "[redacted]" }

// History retention bounds: 30 minutes to 3 days, default 1 day.
const (
	HistoryMinRetention      = 30 * time.Minute
	HistoryMaxRetention      = 72 * time.Hour
	HistoryDefaultRetention  = 24 * time.Hour
	HistoryDefaultRetentionS = "24h"
	HistoryMinItems          = 3
	HistoryMaxItems          = 50
	HistoryDefaultItems      = 10
)

// HistoryRetentions lists the user-facing retention presets, shortest first.
var HistoryRetentions = []string{"30m", "1h", "6h", "12h", "24h", "48h", "72h"}

type History struct {
	Enabled   bool   `yaml:"enabled"`
	Retention string `yaml:"retention,omitempty"`
	MaxItems  int    `yaml:"max_items,omitempty"`
}

type Config struct {
	Removed            []string          `yaml:"removed,omitempty"`
	MembershipVersions map[string]uint64 `yaml:"membership_versions,omitempty"`
	SchemaVersion      int               `yaml:"schema_version,omitempty"`
	Identity           identity.Identity `yaml:"identity,omitempty"`
	Group              struct {
		ID string `yaml:"id"`
	} `yaml:"group,omitempty"`
	Autostart bool `yaml:"autostart"`
	Updates   struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"updates"`
	History History `yaml:"history,omitempty"`
	Device  Device  `yaml:"device"`
	Listen  struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"listen"`
	Security Security `yaml:"security"`
	Sync     struct {
		Enabled bool   `yaml:"enabled"`
		Mode    string `yaml:"mode"`
	} `yaml:"sync"`
	Peers []Peer `yaml:"peers"`
}

func Load(path string) (Config, error) {
	f, e := os.Open(path)
	if e != nil {
		return Config{}, e
	}
	defer f.Close()
	return Parse(f)
}
func Parse(r io.Reader) (Config, error) {
	var c Config
	c.Updates.Enabled = true
	c.Listen.Port = 45873
	c.Sync.Enabled = true
	c.Sync.Mode = "bidirectional"
	c.History.Retention = HistoryDefaultRetentionS
	c.History.MaxItems = HistoryDefaultItems
	b, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return Config{}, errors.New("cannot read configuration or size exceeds 1 MiB")
	}
	d := yaml.NewDecoder(bytes.NewReader(b))
	d.KnownFields(true)
	if e := d.Decode(&c); e != nil {
		return Config{}, errors.New("invalid YAML configuration")
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return Config{}, errors.New("configuration must contain one document")
	}
	return c, c.Validate()
}
func (c Config) Mode() string {
	if !c.Sync.Enabled {
		return "disabled"
	}
	return c.Sync.Mode
}

// HistoryRetention normalizes the stored retention into the allowed range;
// zero or invalid values fall back to the default.
func (c Config) HistoryRetention() time.Duration {
	d, e := time.ParseDuration(c.History.Retention)
	if e != nil || d < HistoryMinRetention || d > HistoryMaxRetention {
		return HistoryDefaultRetention
	}
	return d
}

func (c Config) HistoryMaxItems() int {
	if c.History.MaxItems < HistoryMinItems || c.History.MaxItems > HistoryMaxItems {
		return HistoryDefaultItems
	}
	return c.History.MaxItems
}
func (c Config) ListenAddress() string {
	return net.JoinHostPort(c.Listen.Address, strconv.Itoa(c.Listen.Port))
}
func (c Config) Validate() error {
	if len(c.MembershipVersions) > 256 {
		return errors.New("membership version limit exceeded")
	}
	for id, revision := range c.MembershipVersions {
		if id == "" || len(id) > 128 || revision == 0 || revision >= 1<<63 {
			return errors.New("invalid membership revision")
		}
	}
	if c.SchemaVersion != 0 && c.SchemaVersion != 2 {
		return errors.New("unsupported configuration schema")
	}
	if c.SchemaVersion == 2 {
		if _, err := c.Identity.Private(); err != nil {
			return err
		}
		if c.Group.ID == "" || len(c.Group.ID) > 128 {
			return errors.New("invalid group id")
		}
	}
	if strings.TrimSpace(c.Device.ID) == "" || len(c.Device.ID) > 128 {
		return errors.New("device.id must contain 1-128 bytes")
	}
	if len(c.Device.Name) > 256 || strings.ContainsAny(c.Device.Name, "\n\r\x00") {
		return errors.New("invalid device name")
	}
	if ip := net.ParseIP(c.Listen.Address); ip == nil || ip.IsUnspecified() {
		return errors.New("listen.address must be an explicit IP address")
	}
	if c.Listen.Port < 1 || c.Listen.Port > 65535 {
		return errors.New("invalid listen.port")
	}
	if len(c.Security.Secret) < 32 || c.Security.Secret == "CHANGE_ME" {
		return errors.New("security.secret must contain at least 32 bytes")
	}
	switch c.Sync.Mode {
	case "bidirectional", "disabled", "send-only", "receive-only":
	default:
		return errors.New("invalid sync.mode")
	}
	// Zero values mean "unset" for configs built outside Parse; use defaults.
	if c.History.Retention != "" {
		if d, e := time.ParseDuration(c.History.Retention); e != nil || d < HistoryMinRetention || d > HistoryMaxRetention {
			return errors.New("history.retention must be between 30m and 72h")
		}
	}
	if c.History.MaxItems != 0 && (c.History.MaxItems < HistoryMinItems || c.History.MaxItems > HistoryMaxItems) {
		return errors.New("history.max_items must be between 3 and 50")
	}
	ids := map[string]bool{c.Device.ID: true}
	keys := map[string]bool{c.Identity.PublicKey: true}
	for i, p := range c.Peers {
		if len(p.Name) > 256 || strings.ContainsAny(p.Name, "\n\r\x00") {
			return fmt.Errorf("peer %d: invalid name", i)
		}
		if p.ID == "" || len(p.ID) > 128 || ids[p.ID] {
			return fmt.Errorf("peer %d: invalid or duplicate id", i)
		}
		ids[p.ID] = true
		if c.SchemaVersion == 2 && !p.Legacy {
			if _, err := identity.Public(p.PublicKey); err != nil {
				return fmt.Errorf("peer %d: invalid public key", i)
			}
			if keys[p.PublicKey] {
				return errors.New("duplicate device public key")
			}
			keys[p.PublicKey] = true
		}
		if p.LegacySecret != "" && len(p.LegacySecret) < 32 {
			return errors.New("invalid legacy peer secret")
		}
		if p.Port < 1 || p.Port > 65535 || p.Address == "" || strings.ContainsAny(p.Address, "/\\?#@ \t\r\n") {
			return fmt.Errorf("peer %d: invalid address or port", i)
		}
	}
	if len(c.Peers) > 64 {
		return errors.New("maximum 64 peers")
	}
	return nil
}
