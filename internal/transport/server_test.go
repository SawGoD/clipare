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
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type receiver struct{ count int }

func (r *receiver) Receive(context.Context, clipsync.Message) error { r.count++; return nil }
func TestAuthenticatedEndpoint(t *testing.T) {
	var c config.Config
	c.Device.ID = "b"
	c.Peers = []config.Peer{{ID: "a"}}
	c.Sync.Enabled = true
	c.Sync.Mode = "bidirectional"
	secret := security.StaticSecret("test secret")
	rec := &receiver{}
	var logs bytes.Buffer
	h := Handler(c, secret, rec, slog.New(slog.NewTextHandler(&logs, nil)))
	m, _ := clipsync.NewMessage("a", "PRIVATE_CLIPBOARD_TEXT")
	body, _ := json.Marshal(m)
	check := func(b []byte, signed bool, want int) {
		t.Helper()
		r := httptest.NewRequest("POST", "/api/v1/clipboard", bytes.NewReader(b))
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		r.Header.Set(security.TimestampHeader, ts)
		if signed {
			r.Header.Set(security.SignatureHeader, security.Sign(secret.Secret(), ts, b))
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("got %d want %d", w.Code, want)
		}
	}
	check(body, false, 401)
	check(body, true, 200)
	m.Source = "stranger"
	b, _ := json.Marshal(m)
	check(b, true, 403)
	check(append(body, []byte(" {}")...), true, 400)
	m.Source = "a"
	m.Data = strings.Repeat("x", clipsync.MaxText+1)
	b, _ = json.Marshal(m)
	check(b, true, 400)
	if rec.count != 1 || strings.Contains(logs.String(), "PRIVATE_CLIPBOARD_TEXT") {
		t.Fatal("leak or invalid delivery")
	}
	s := httptest.NewServer(h)
	defer s.Close()
	client := NewClient(secret)
	defer client.Close()
	health, e := client.Health(context.Background(), s.URL)
	if e != nil || health.Device != "b" {
		t.Fatal(health, e)
	}
	bad := NewClient(security.StaticSecret("wrong"))
	defer bad.Close()
	if _, e = bad.Health(context.Background(), s.URL); e == nil {
		t.Fatal("invalid HMAC accepted")
	}
}
func TestPeerIsolationAndCancellation(t *testing.T) {
	delivered := make(chan struct{}, 1)
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		if r.Method == "GET" {
			io.WriteString(w, `{"status":"ok","device":"fast"}`)
		} else {
			delivered <- struct{}{}
		}
	}))
	defer fast.Close()
	// Use URL host/port parsed by net/http rather than a live network dependency.
	peer := func(id, url string) config.Peer {
		r, _ := http.NewRequest("GET", url, nil)
		port, _ := strconv.Atoi(r.URL.Port())
		return config.Peer{ID: id, Address: r.URL.Hostname(), Port: port}
	}
	ctx, cancel := context.WithCancel(context.Background())
	client := NewClient(security.StaticSecret("secret"))
	defer client.Close()
	w := StartWorkers(ctx, client, []config.Peer{peer("slow", slow.URL), peer("fast", fast.URL)}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	m, _ := clipsync.NewMessage("a", "text")
	w.Broadcast(m)
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("slow peer blocks fast peer")
	}
	cancel()
	done := make(chan struct{})
	go func() { w.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers failed to stop")
	}
}
