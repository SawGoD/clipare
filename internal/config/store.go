package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go.yaml.in/yaml/v3"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func DefaultPath() (string, error) {
	p, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(p, "Clipare", "config.yaml"), nil
}
func Addresses() []string {
	var result []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ip, _, e := net.ParseCIDR(a.String())
		if e == nil && ip.To4() != nil && !ip.IsLoopback() {
			result = append(result, ip.String())
		}
	}
	return result
}
func NewSecret() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
func Default() (Config, error) {
	var c Config
	id, e := NewSecret()
	if e != nil {
		return c, e
	}
	c.Device.ID = id[:16]
	c.Device.Name, _ = os.Hostname()
	c.Listen.Port = 45873
	c.Sync.Enabled = true
	c.Sync.Mode = "bidirectional"
	c.Listen.Address = "127.0.0.1"
	for _, a := range Addresses() {
		ip := net.ParseIP(a).To4()
		if ip[0] == 100 && ip[1] >= 64 && ip[1] <= 127 {
			c.Listen.Address = a
			break
		}
	}
	c.Security.Secret, e = NewSecret()
	return c, e
}

// Save atomically replaces the config, with owner-only permissions on Unix.
func Save(path string, c Config) error {
	if e := c.Validate(); e != nil {
		return e
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return errors.New("cannot create configuration directory")
	}
	b, e := yaml.Marshal(c)
	if e != nil {
		return errors.New("cannot encode configuration")
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".clipare-*.tmp")
	if e != nil {
		return errors.New("cannot create configuration")
	}
	temp := f.Name()
	defer os.Remove(temp)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil {
		return errors.New("cannot write configuration")
	}
	if e = replaceFile(temp, path); e != nil {
		return errors.New("cannot replace configuration")
	}
	return nil
}

const CodePrefix = "clipare1:"

type connection struct {
	Peer   Peer   `json:"peer"`
	Secret string `json:"secret"`
}

func ConnectionCode(c Config) string {
	b, _ := json.Marshal(connection{Peer{ID: c.Device.ID, Name: c.Device.Name, Address: c.Listen.Address, Port: c.Listen.Port}, c.Security.Secret})
	return CodePrefix + base64.RawURLEncoding.EncodeToString(b)
}
func AddConnection(c Config, code string) (Config, error) {
	if len(code) > 4096 || !strings.HasPrefix(code, CodePrefix) {
		return c, errors.New("Некорректный код подключения")
	}
	b, e := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(code, CodePrefix))
	if e != nil {
		return c, errors.New("Некорректный код подключения")
	}
	var v connection
	if json.Unmarshal(b, &v) != nil {
		return c, errors.New("Некорректный код подключения")
	}
	if len(c.Peers) > 0 && c.Security.Secret != v.Secret {
		return c, errors.New("Общий ключ отличается: используйте код из существующей группы устройств")
	}
	if v.Peer.ID == c.Device.ID {
		return c, errors.New("Это код этого компьютера")
	}
	c.Security.Secret = v.Secret
	found := false
	c.Peers = append([]Peer(nil), c.Peers...)
	for i, p := range c.Peers {
		if p.ID == v.Peer.ID {
			c.Peers[i] = v.Peer
			found = true
			break
		}
	}
	if !found {
		c.Peers = append(c.Peers, v.Peer)
	}
	if c.Validate() != nil {
		return c, errors.New("Код содержит неверный адрес, порт или ключ")
	}
	return c, nil
}
func Platform() string { return runtime.GOOS }
