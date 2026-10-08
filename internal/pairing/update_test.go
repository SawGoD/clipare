package pairing

import (
	"clipare/internal/config"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateSuspendsOnlyIdlePairing(t *testing.T) {
	c, _ := config.Default()
	s := NewService(filepath.Join(t.TempDir(), "config.yaml"), c)
	if !s.SuspendPairing() || !s.disabled.Load() {
		t.Fatal("not suspended")
	}
	if s.SuspendPairing() {
		t.Fatal("double reservation")
	}
	s.ResumePairing()
	s.outgoing.Store(true)
	if s.SuspendPairing() {
		t.Fatal("active outgoing")
	}
	s.outgoing.Store(false)
	h, _ := NewHandshake(c)
	id, _ := NewID()
	req := Request{Session: id, Commitment: Commitment(h.Hello), Expires: s.sessions.now().Add(time.Minute).Unix()}
	if _, e := s.sessions.Begin(c, req); e != nil {
		t.Fatal(e)
	}
	if s.SuspendPairing() {
		t.Fatal("active incoming")
	}
}
