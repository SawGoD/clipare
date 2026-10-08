package pairing

import (
	"bytes"
	"clipare/internal/config"
	"testing"
	"time"
)

func TestCommittedHandshake(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	ha, _ := NewHandshake(a)
	sessions := NewSessions()
	id, _ := NewID()
	request := Request{Session: id, Commitment: Commitment(ha.Hello), Expires: ha.Hello.Expires}
	ch, e := sessions.Begin(b, request)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = sessions.Begin(b, request); e == nil {
		t.Fatal("duplicate session")
	}
	ka, sas, e := ha.Keys(ch.Hello, id, ch.Group, ch.Expires, true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = sessions.Reveal(Exchange{Session: id, Hello: ha.Hello}); e == nil {
		t.Fatal("notification without initiator key proof")
	}
	confirmation, e := sessions.Reveal(Exchange{Session: id, Hello: ha.Hello, Proof: proof(ka, id, "reveal")})
	if e != nil || !verifyProof(ka, id, "exchange", confirmation.Proof) {
		t.Fatal("key confirmation")
	}
	snap, ok := sessions.Snapshot()
	if !ok || snap.SAS != sas {
		t.Fatal("SAS mismatch")
	}
	finish := Confirmation{id, proof(ka, id, "finish")}
	if _, e = sessions.Finish(finish); e == nil {
		t.Fatal("approval bypass")
	}
	if e = sessions.Decide(id, true); e != nil {
		t.Fatal(e)
	}
	if _, e = sessions.Finish(finish); e != nil {
		t.Fatal(e)
	}
	if _, e = sessions.Finish(finish); e == nil {
		t.Fatal("replay")
	}
	if _, e = sessions.Begin(b, request); e == nil {
		t.Fatal("replayed request")
	}
	hb, _ := NewHandshake(b)
	h2, _ := NewHandshake(a)
	id2, _ := NewID()
	k2, _, _ := h2.Keys(hb.Hello, id2, ch.Group, ch.Expires, true)
	if bytes.Equal(ka, k2) {
		t.Fatal("session keys reused")
	}
}
func TestExpiryAndCommitment(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	h, _ := NewHandshake(a)
	s := NewSessions()
	id, _ := NewID()
	ch, _ := s.Begin(b, Request{Session: id, Commitment: Commitment(h.Hello), Expires: h.Hello.Expires})
	bad := h.Hello
	bad.Name = "forged"
	if _, e := s.Reveal(Exchange{Session: id, Hello: bad}); e == nil {
		t.Fatal("commitment bypass")
	}
	s.now = func() time.Time { return time.Unix(ch.Expires, 0) }
	if _, e := s.Reveal(Exchange{Session: id, Hello: h.Hello}); e == nil {
		t.Fatal("expired session")
	}
}
