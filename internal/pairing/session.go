package pairing

import (
	"clipare/internal/config"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Request struct {
	Session    string `json:"session"`
	Commitment string `json:"commitment"`
	Expires    int64  `json:"expires"`
}
type Challenge struct {
	Session string `json:"session"`
	Hello   Hello  `json:"hello"`
	Group   string `json:"group"`
	Expires int64  `json:"expires"`
}
type Exchange struct {
	Session string `json:"session"`
	Hello   Hello  `json:"hello"`
	Proof   string `json:"proof"`
}
type Confirmation struct {
	Session string `json:"session"`
	Proof   string `json:"proof"`
}
type State string

const (
	Waiting  State = "waiting"
	Pending  State = "pending"
	Approved State = "approved"
	Rejected State = "rejected"
	Complete State = "complete"
)

type Snapshot struct {
	Session string
	Name    string
	SAS     string
	State   State
	Expires int64
}
type record struct {
	requestExpires int64
	challenge      Challenge
	commitment     string
	handshake      *Handshake
	remote         Hello
	key            []byte
	sas            string
	state          State
}

func (*record) String() string   { return "[redacted pairing session]" }
func (*record) GoString() string { return "[redacted pairing session]" }

type Sessions struct {
	mu     sync.Mutex
	active *record
	used   map[string]time.Time
	now    func() time.Time
}

func NewSessions() *Sessions { return &Sessions{used: map[string]time.Time{}, now: time.Now} }
func (s *Sessions) Available() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup()
	return s.active == nil
}
func NewID() (string, error) {
	b := make([]byte, 32)
	_, e := rand.Read(b)
	return hex.EncodeToString(b), e
}
func (s *Sessions) cleanup() {
	now := s.now()
	for id, t := range s.used {
		if !now.Before(t) {
			delete(s.used, id)
		}
	}
	if s.active != nil && now.Unix() >= s.active.challenge.Expires {
		s.active = nil
	}
}
func (s *Sessions) Begin(c config.Config, r Request) (Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup()
	if r.Expires <= s.now().Unix() || r.Expires > s.now().Add(180*time.Second).Unix() {
		return Challenge{}, ErrExpired
	}
	id, e := hex.DecodeString(r.Session)
	if e != nil || len(id) != 32 {
		return Challenge{}, ErrInvalid
	}
	commit, e := hex.DecodeString(r.Commitment)
	if e != nil || len(commit) != 32 {
		return Challenge{}, ErrInvalid
	}
	if s.active != nil || len(s.used) >= 256 {
		return Challenge{}, ErrApproval
	}
	if _, seen := s.used[r.Session]; seen {
		return Challenge{}, ErrInvalid
	}
	h, e := NewHandshake(c)
	if e != nil {
		return Challenge{}, e
	}
	expires := s.now().Add(120 * time.Second).Unix()
	if r.Expires < expires {
		expires = r.Expires
	}
	ch := Challenge{r.Session, h.Hello, c.Group.ID, expires}
	s.used[r.Session] = s.now().Add(5 * time.Minute)
	s.active = &record{challenge: ch, commitment: r.Commitment, handshake: h, state: Waiting, requestExpires: r.Expires}
	return ch, nil
}
func (s *Sessions) Reveal(r Exchange) (Confirmation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup()
	a := s.active
	if a == nil || a.challenge.Session != r.Session {
		return Confirmation{}, ErrExpired
	}
	if a.state != Waiting || r.Hello.Expires != a.requestExpires || Commitment(r.Hello) != a.commitment {
		return Confirmation{}, ErrInvalid
	}
	k, sas, e := a.handshake.Keys(r.Hello, r.Session, a.challenge.Group, a.challenge.Expires, false)
	if e != nil {
		return Confirmation{}, e
	}
	if !verifyProof(k, r.Session, "reveal", r.Proof) {
		return Confirmation{}, ErrInvalid
	}
	a.remote = r.Hello
	a.key = k
	a.sas = sas
	a.state = Pending
	return Confirmation{r.Session, proof(k, r.Session, "exchange")}, nil
}
func (s *Sessions) Snapshot() (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup()
	a := s.active
	if a == nil || a.state == Waiting {
		return Snapshot{}, false
	}
	return Snapshot{a.challenge.Session, a.remote.Name, a.sas, a.state, a.challenge.Expires}, true
}
func (s *Sessions) Decide(id string, allow bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup()
	a := s.active
	if a == nil || a.challenge.Session != id {
		return ErrExpired
	}
	if a.state != Pending {
		return ErrInvalid
	}
	if allow {
		a.state = Approved
	} else {
		a.state = Rejected
	}
	return nil
}
func (s *Sessions) Auth(r Confirmation, action string) (Hello, State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup()
	a := s.active
	if a == nil || a.challenge.Session != r.Session {
		return Hello{}, "", ErrExpired
	}
	if !verifyProof(a.key, r.Session, action, r.Proof) {
		return Hello{}, "", ErrInvalid
	}
	return a.remote, a.state, nil
}
func (s *Sessions) Finish(r Confirmation) (Hello, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup()
	a := s.active
	if a == nil || a.challenge.Session != r.Session {
		return Hello{}, ErrExpired
	}
	if a.state != Approved {
		return Hello{}, ErrApproval
	}
	if !verifyProof(a.key, r.Session, "finish", r.Proof) {
		return Hello{}, ErrInvalid
	}
	remote := a.remote
	s.active = nil
	return remote, nil
}
