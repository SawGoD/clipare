package pairing

import (
	"bytes"
	"clipare/internal/config"
	"clipare/internal/discovery"
	"clipare/internal/peers"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestServiceApprovalAndUnknownMembership(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	a.Listen.Address = "100.64.0.1"
	b.Listen.Address = "100.64.0.2"
	s := NewService(filepath.Join(t.TempDir(), "config.yaml"), b)
	h, _ := NewHandshake(a)
	id, _ := NewID()
	ch, e := s.sessions.Begin(b, Request{id, Commitment(h.Hello)})
	if e != nil {
		t.Fatal(e)
	}
	key, _, _ := h.Keys(ch.Hello, id, ch.Group, ch.Expires, true)
	s.sessions.Reveal(Exchange{id, h.Hello})
	if len(s.Config().Peers) != 0 {
		t.Fatal("peer admitted before approval")
	}
	if e = s.Decide(id, true); e != nil {
		t.Fatal(e)
	}
	if len(s.Config().Peers) != 1 {
		t.Fatal("approved peer not saved")
	}
	call := func(path string, v any, want int) []byte {
		t.Helper()
		s.limiter = &discovery.Limiter{Interval: time.Nanosecond}
		body, _ := json.Marshal(v)
		r := httptest.NewRequest("POST", path, bytes.NewReader(body))
		r.RemoteAddr = "100.64.0.1:1234"
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("status %d want %d", w.Code, want)
		}
		return w.Body.Bytes()
	}
	var status Status
	json.Unmarshal(call("/api/v1/pair/status", Confirmation{id, proof(key, id, "status")}, 200), &status)
	if status.State != Approved || !verifyProof(key, id, statusAction(status), status.Proof) {
		t.Fatal("invalid approval proof")
	}
	if e = s.Adopt(peers.Membership{Group: b.Group.ID, Members: []peers.Member{peers.Local(a)}}, "unknown"); e == nil {
		t.Fatal("unknown group sponsor")
	}
	call("/api/v1/group/membership", status.Membership, 401)
	call("/api/v1/pair/finish", Confirmation{id, proof(key, id, "finish")}, 200)
	call("/api/v1/pair/finish", Confirmation{id, proof(key, id, "finish")}, 400)
}
