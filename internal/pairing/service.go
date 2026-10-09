package pairing

import (
	"bytes"
	"clipare"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/peers"
	"clipare/internal/security"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Service struct {
	mu       sync.Mutex
	pairGate sync.Mutex
	c        config.Config
	path     string
	sessions *Sessions
	limiter  *discovery.Limiter
	discover http.Handler
	Updates  chan config.Config
	client   *http.Client // injectable transport for protocol integration tests
	outgoing atomic.Bool
	disabled atomic.Bool
	wake     chan struct{}
}

func (*Service) String() string   { return "[redacted pairing service]" }
func (*Service) GoString() string { return "[redacted pairing service]" }

func NewService(path string, c config.Config) *Service {
	s := &Service{c: c, path: path, sessions: NewSessions(), limiter: &discovery.Limiter{Interval: time.Second}, Updates: make(chan config.Config, 1), wake: make(chan struct{}, 1)}
	s.discover = discovery.Handler(func() discovery.Info {
		v := s.Config()
		return discovery.Info{Protocol: 1, App: "clipare", Version: clipare.Version(), DeviceID: v.Device.ID, DeviceName: v.Device.Name, PublicKey: v.Identity.PublicKey, PairingAvailable: !s.disabled.Load() && !s.outgoing.Load() && s.sessions.Available()}
	})
	return s
}
func (s *Service) Config() config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.c
	c.Peers = append([]config.Peer(nil), c.Peers...)
	c.Removed = append([]string(nil), c.Removed...)
	c.MembershipVersions = peers.CloneVersions(c.MembershipVersions)
	return c
}

// SaveSettings preserves authenticated membership received during a user edit.
func (s *Service) SaveSettings(next config.Config) (config.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if next.Group.ID != s.c.Group.ID {
		return next, errors.New("Состав группы изменился. Откройте настройки заново")
	}
	if discovery.NetBirdAddress(s.c.Listen.Address) {
		merged, err := peers.Merge(next, peers.Export(s.c))
		if err != nil {
			return next, ErrInvalid
		}
		next = merged
	}
	if err := s.saveLocked(next); err != nil {
		return next, err
	}
	return next, nil
}
func (s *Service) DisablePairing() { s.disabled.Store(true) }

// SuspendPairing reserves the idle pairing lifecycle while installing updates.
// The gate prevents a request slipping between the idle check and disable flag.
func (s *Service) SuspendPairing() bool {
	s.pairGate.Lock()
	defer s.pairGate.Unlock()
	if s.disabled.Load() || s.outgoing.Load() || !s.sessions.Available() {
		return false
	}
	s.disabled.Store(true)
	return true
}
func (s *Service) ResumePairing() { s.disabled.Store(false) }
func (s *Service) saveLocked(c config.Config) error {
	if reflect.DeepEqual(c, s.c) {
		return nil
	}
	if err := config.Save(s.path, c); err != nil {
		return errors.New("Не удалось сохранить подключение")
	}
	s.c = c
	select {
	case s.wake <- struct{}{}:
	default:
	}
	select {
	case s.Updates <- c:
	default:
		select {
		case <-s.Updates:
		default:
		}
		s.Updates <- c
	}
	return nil
}
func (s *Service) Snapshot() (Snapshot, bool) { return s.sessions.Snapshot() }
func (s *Service) Decide(id string, allow bool) error {
	return s.sessions.decideWith(id, allow, s.persistPair)
}
func (s *Service) persistPair(remote Hello, group string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := peers.Reinstate(s.c, peers.Membership{}, remote.ID)
	if err != nil {
		return err
	}
	if c.Group.ID != group && len(c.Peers) > 0 {
		return ErrGroupMerge
	}
	c.Group.ID = group
	v := peers.Export(c)
	// A peer may have lost its local relationship while we still trust its
	// identity. Approval repairs metadata, not a second membership entry.
	found := false
	for i, existing := range v.Members {
		if existing.ID == remote.ID {
			if existing.PublicKey != remote.PublicKey {
				return ErrIdentityChanged
			}
			v.Members[i] = member(remote)
			found = true
			break
		}
	}
	if !found {
		v.Members = append(v.Members, member(remote))
	}
	next, err := peers.Merge(c, v)
	if err != nil {
		return ErrInvalid
	}
	if err = s.saveLocked(next); err != nil {
		return err
	}
	return nil
}
func member(h Hello) peers.Member {
	return peers.Member{ID: h.ID, Name: h.Name, IP: h.IP, Port: h.Port, PublicKey: h.PublicKey}
}

type Status struct {
	State      State            `json:"state"`
	Membership peers.Membership `json:"membership"`
	Proof      string           `json:"proof"`
}

func statusAction(v Status) string {
	v.Proof = ""
	b, _ := json.Marshal(v)
	return "status/" + string(b)
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}
func decode(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return ErrInvalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return ErrInvalid
	}
	return nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/discovery" {
		s.discover.ServeHTTP(w, r)
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", 405)
		return
	}
	if !s.limiter.Allow(r.RemoteAddr) {
		http.Error(w, "rate limited", 429)
		return
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !discovery.NetBirdAddress(ip) {
		http.Error(w, "NetBird source required", 403)
		return
	}
	limit := int64(32 << 10)
	if r.URL.Path == "/api/v1/group/membership" {
		limit = 128 << 10
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		http.Error(w, "request too large", 413)
		return
	}
	fail := func(err error) {
		code := 400
		if errors.Is(err, ErrExpired) {
			code = 410
		}
		if errors.Is(err, ErrApproval) {
			code = 409
		}
		if errors.Is(err, ErrGroupMerge) {
			code = 412
		}
		if errors.Is(err, ErrIdentityChanged) {
			code = 422
		}
		if errors.Is(err, ErrRemoved) {
			code = 423
		}
		http.Error(w, "pairing unavailable", code)
	}
	switch r.URL.Path {
	case "/api/v1/group/upgrade":
		s.serveUpgrade(w, r, b, ip)
	case "/api/v1/pair/request":
		s.pairGate.Lock()
		defer s.pairGate.Unlock()
		if s.disabled.Load() || s.outgoing.Load() {
			fail(ErrApproval)
			return
		}
		var v Request
		if decode(b, &v) != nil {
			fail(ErrInvalid)
			return
		}
		ch, e := s.sessions.Begin(s.Config(), v)
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, ch)
	case "/api/v1/pair/exchange":
		var v Exchange
		if decode(b, &v) != nil || v.Hello.IP != ip || member(v.Hello).Validate() != nil {
			fail(ErrInvalid)
			return
		}
		ch, e := s.sessions.Reveal(v)
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, ch)
	case "/api/v1/pair/status":
		var v Confirmation
		if decode(b, &v) != nil {
			fail(ErrInvalid)
			return
		}
		remote, state, e := s.sessions.Auth(v, "status")
		if e != nil || remote.IP != ip {
			fail(ErrInvalid)
			return
		}
		out := Status{State: state}
		if state == Approved {
			out.Membership = peers.Export(s.Config())
		}
		s.sessions.mu.Lock()
		a := s.sessions.active
		if a == nil {
			s.sessions.mu.Unlock()
			fail(ErrExpired)
			return
		}
		out.Proof = proof(a.key, v.Session, statusAction(out))
		if state == Rejected {
			s.sessions.active = nil
		}
		s.sessions.mu.Unlock()
		writeJSON(w, out)
	case "/api/v1/pair/confirm":
		var v Confirmation
		if decode(b, &v) != nil {
			fail(ErrInvalid)
			return
		}
		remote, _, e := s.sessions.Auth(v, "confirm")
		if e != nil || remote.IP != ip {
			fail(ErrInvalid)
			return
		}
		if e = s.sessions.confirmWith(v, s.persistPair); e != nil {
			fail(e)
			return
		}
		out, e := s.sessions.Response(v.Session, "confirm-response")
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, out)
	case "/api/v1/pair/finish":
		var v Confirmation
		if decode(b, &v) != nil {
			fail(ErrInvalid)
			return
		}
		remote, _, e := s.sessions.Auth(v, "finish")
		if e != nil || remote.IP != ip {
			fail(ErrInvalid)
			return
		}
		if _, e = s.sessions.Finish(v); e != nil {
			fail(e)
			return
		}
		writeJSON(w, struct {
			Status string `json:"status"`
		}{"ok"})
	case "/api/v1/pair/cancel":
		var v Confirmation
		if decode(b, &v) != nil {
			fail(ErrInvalid)
			return
		}
		remote, _, e := s.sessions.Auth(v, "cancel")
		if e != nil || remote.IP != ip {
			fail(ErrInvalid)
			return
		}
		s.sessions.mu.Lock()
		s.sessions.active = nil
		s.sessions.mu.Unlock()
		writeJSON(w, struct {
			Status string `json:"status"`
		}{"ok"})
	case "/api/v1/group/membership":
		c := s.Config()
		source := r.Header.Get(security.SourceHeader)
		keys := security.NewPeerKeys(c)
		key, e := keys.KeyForPeer(source)
		if e != nil || keys.Legacy(source) || !security.Verify(key, r.Header.Get(security.TimestampHeader), r.Header.Get(security.SignatureHeader), security.AuthenticatedData(r.Method, r.URL.Path, source, b), time.Now()) {
			http.Error(w, "unauthorized", 401)
			return
		}
		var v peers.Membership
		if decode(b, &v) != nil {
			fail(ErrInvalid)
			return
		}
		s.mu.Lock()
		next, e := peers.Merge(s.c, v)
		if e == nil {
			for j := range next.Peers {
				if next.Peers[j].ID == source {
					next.Peers[j].LegacySecret = ""
				}
			}
			e = s.saveLocked(next)
		}
		s.mu.Unlock()
		if e != nil {
			fail(e)
			return
		}
		writeJSON(w, struct {
			Status string `json:"status"`
		}{"ok"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Service) Adopt(v peers.Membership, expected string, fqdn ...string) error {
	return s.adopt(v, expected, false, fqdn...)
}
func (s *Service) adopt(v peers.Membership, expected string, approved bool, fqdn ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.c
	if approved {
		var err error
		c, err = peers.Reinstate(c, v, expected)
		if err != nil {
			return err
		}
	}
	// Joining a different established group requires a separate explicit merge
	// UX. Never silently discard an existing group's relationships.
	if c.Group.ID != v.Group {
		for _, p := range c.Peers {
			if !p.Legacy {
				return errors.New("Это устройство уже подключён к другой группе")
			}
		}
	}
	known := false
	for _, m := range v.Members {
		if m.ID == expected {
			known = true
		}
	}
	if !known {
		return ErrInvalid
	}
	c.Group.ID = v.Group
	if len(fqdn) > 0 && fqdn[0] != "" {
		for j := range v.Members {
			if v.Members[j].ID == expected {
				v.Members[j].FQDN = fqdn[0]
			}
		}
	}
	next, err := peers.Merge(c, v)
	if err != nil {
		return ErrInvalid
	}
	// A tombstone (local or supplied by the group) must not produce a
	// successful pairing with no relationship actually saved.
	for _, p := range next.Peers {
		if p.ID == expected {
			return s.saveLocked(next)
		}
	}
	return ErrRemoved
}

// Propagate periodically reconciles metadata without depending on online peers
// during approval. Requests are sequential and cancellable, with no retry queue.
func (s *Service) Propagate(ctx context.Context) {
	client := pairClient()
	defer client.CloseIdleConnections()
	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.wake:
		}
		c := s.Config()
		body, _ := json.Marshal(peers.Export(c))
		keys := security.NewPeerKeys(c)
		for _, p := range c.Peers {
			if ctx.Err() != nil {
				return
			}
			if p.Legacy {
				s.tryUpgrade(ctx, client, c, p)
				continue
			}
			key, e := keys.KeyForPeer(p.ID)
			if e != nil {
				continue
			}
			ip := p.LastKnownIP
			if discovery.NetBirdAddress(p.Address) {
				ip = p.Address
			}
			if !discovery.NetBirdAddress(ip) {
				continue
			}
			r, e := http.NewRequestWithContext(ctx, "POST", "http://"+net.JoinHostPort(ip, strconv.Itoa(p.Port))+"/api/v1/group/membership", bytes.NewReader(body))
			if e != nil {
				continue
			}
			ts := strconv.FormatInt(time.Now().Unix(), 10)
			r.Header.Set(security.SourceHeader, c.Device.ID)
			r.Header.Set(security.TimestampHeader, ts)
			r.Header.Set(security.SignatureHeader, security.Sign(key, ts, security.AuthenticatedData(r.Method, r.URL.Path, c.Device.ID, body)))
			res, e := client.Do(r)
			if e == nil {
				io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
				res.Body.Close()
			}
		}
	}
}
