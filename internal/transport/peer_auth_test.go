package transport

import (
	"bytes"
	"clipare/internal/config"
	"clipare/internal/security"
	clipsync "clipare/internal/sync"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestPeerCannotImpersonateAndRemovedPeer(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	c, _ := config.Default()
	b.Group = a.Group
	c.Group = a.Group
	b.Peers = []config.Peer{{ID: a.Device.ID, PublicKey: a.Identity.PublicKey}, {ID: c.Device.ID, PublicKey: c.Identity.PublicKey}}
	a.Peers = []config.Peer{{ID: b.Device.ID, PublicKey: b.Identity.PublicKey}}
	rec := &receiver{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := Handler(b, security.NewPeerKeys(b), rec, log)
	m, _ := clipsync.NewMessage(a.Device.ID, "secret text")
	body, _ := json.Marshal(m)
	key, _ := security.NewPeerKeys(a).KeyForPeer(b.Device.ID)
	request := func(source string, body []byte, want int) {
		t.Helper()
		r := httptest.NewRequest("POST", "/api/v1/clipboard", bytes.NewReader(body))
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		r.Header.Set(security.SourceHeader, source)
		r.Header.Set(security.TimestampHeader, ts)
		r.Header.Set(security.SignatureHeader, security.Sign(key, ts, security.AuthenticatedData("POST", r.URL.Path, source, body)))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("got %d want %d", w.Code, want)
		}
	}
	request(a.Device.ID, body, 200)
	m.Source = c.Device.ID
	forged, _ := json.Marshal(m)
	request(c.Device.ID, forged, 401)
	request(a.Device.ID, forged, 403)
	b.Peers = b.Peers[1:]
	h = Handler(b, security.NewPeerKeys(b), rec, log)
	request(a.Device.ID, body, 403)
	r := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.RemoteAddr = "100.64.0.99:1234"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r.Header.Set(security.TimestampHeader, ts)
	r.Header.Set(security.SignatureHeader, security.Sign([]byte(b.Security.Secret), ts, nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("remote global-secret health accepted for paired-only group")
	}
	// Old and new senders coexist. Old health and clipboard keep their wire format.
	b.Peers = append(b.Peers, config.Peer{ID: a.Device.ID, Legacy: true, LegacySecret: a.Security.Secret})
	server := httptest.NewServer(Handler(b, security.NewPeerKeys(b), rec, log))
	defer server.Close()
	old := NewClient(security.StaticSecret(a.Security.Secret))
	defer old.Close()
	m.Source = a.Device.ID
	if err := old.Send(context.Background(), server.URL, m); err != nil {
		t.Fatal(err)
	}
}
