package pairing

import (
	"bytes"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/peers"
	"clipare/internal/security"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

type upgrade struct {
	Group  string       `json:"group"`
	Member peers.Member `json:"member"`
	Nonce  string       `json:"nonce"`
}
type upgraded struct {
	Upgrade   upgrade `json:"upgrade"`
	Signature string  `json:"signature"`
}

func legacyKey(c config.Config, id string) ([]byte, bool) {
	for _, p := range c.Peers {
		if p.ID == id {
			if p.LegacySecret != "" {
				return []byte(p.LegacySecret), true
			}
			if p.Legacy {
				return []byte(c.Security.Secret), true
			}
		}
	}
	return nil, false
}
func (s *Service) upgradePeer(v upgrade, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.c
	if v.Group != c.Group.ID || v.Member.ID != source || v.Member.Validate() != nil {
		return ErrInvalid
	}
	c.Peers = append([]config.Peer(nil), c.Peers...)
	for j, p := range c.Peers {
		if p.ID != source {
			continue
		}
		if !p.Legacy && p.PublicKey != v.Member.PublicKey {
			return ErrInvalid
		}
		c.Peers[j].Legacy = false
		c.Peers[j].PublicKey = v.Member.PublicKey
		c.Peers[j].LastKnownIP = v.Member.IP
		return s.saveLocked(c)
	}
	return ErrInvalid
}
func (s *Service) serveUpgrade(w http.ResponseWriter, r *http.Request, body []byte, ip string) {
	c := s.Config()
	source := r.Header.Get(security.SourceHeader)
	key, ok := legacyKey(c, source)
	ts := r.Header.Get(security.TimestampHeader)
	if !ok || !security.Verify(key, ts, r.Header.Get(security.SignatureHeader), security.AuthenticatedData(r.Method, r.URL.Path, source, body), time.Now()) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var v upgrade
	if decode(body, &v) != nil || len(v.Nonce) != 64 || v.Member.IP != ip {
		http.Error(w, "invalid upgrade", 400)
		return
	}
	if err := s.upgradePeer(v, source); err != nil {
		http.Error(w, "invalid upgrade", 400)
		return
	}
	out := upgrade{Group: c.Group.ID, Member: peers.Local(c), Nonce: v.Nonce}
	b, _ := json.Marshal(out)
	writeJSON(w, upgraded{out, security.Sign(key, ts, security.AuthenticatedData("RESPONSE", r.URL.Path, c.Device.ID, b))})
}
func (s *Service) tryUpgrade(ctx context.Context, client *http.Client, c config.Config, p config.Peer) {
	if !p.Legacy {
		return
	}
	key, ok := legacyKey(c, p.ID)
	if !ok {
		return
	}
	nonce, err := NewID()
	if err != nil {
		return
	}
	v := upgrade{c.Group.ID, peers.Local(c), nonce}
	body, _ := json.Marshal(v)
	// Legacy metadata may only have FQDN. Resolve it, but never probe a public IP.
	lookup, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	ips, err := net.DefaultResolver.LookupIPAddr(lookup, p.Address)
	cancel()
	if err != nil {
		return
	}
	ip := ""
	for _, v := range ips {
		if discovery.NetBirdAddress(v.IP.String()) {
			ip = v.IP.String()
			break
		}
	}
	if ip == "" {
		return
	}
	r, err := http.NewRequestWithContext(ctx, "POST", "http://"+net.JoinHostPort(ip, strconv.Itoa(p.Port))+"/api/v1/group/upgrade", bytes.NewReader(body))
	if err != nil {
		return
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r.Header.Set(security.SourceHeader, c.Device.ID)
	r.Header.Set(security.TimestampHeader, ts)
	r.Header.Set(security.SignatureHeader, security.Sign(key, ts, security.AuthenticatedData(r.Method, r.URL.Path, c.Device.ID, body)))
	res, err := client.Do(r)
	if err != nil {
		return
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4097))
	if err != nil || len(b) > 4096 || res.StatusCode != 200 {
		return
	}
	var out upgraded
	if decode(b, &out) != nil || out.Upgrade.Nonce != nonce || out.Upgrade.Member.ID != p.ID || out.Upgrade.Group != c.Group.ID {
		return
	}
	signed, _ := json.Marshal(out.Upgrade)
	if !security.Verify(key, ts, out.Signature, security.AuthenticatedData("RESPONSE", r.URL.Path, p.ID, signed), time.Now()) {
		return
	}
	_ = s.upgradePeer(out.Upgrade, p.ID)
}
