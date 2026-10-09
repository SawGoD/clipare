package transport

import (
	"clipare/internal/config"
	"clipare/internal/security"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestAuthenticatedHealthPlatform(t *testing.T) {
	a, _ := config.Default()
	b, _ := config.Default()
	b.Group = a.Group
	a.Listen.Address = "127.0.0.1"
	a.Peers = []config.Peer{{ID: b.Device.ID, PublicKey: b.Identity.PublicKey}}
	b.Peers = []config.Peer{{ID: a.Device.ID, PublicKey: a.Identity.PublicKey}}
	handler := Handler(b, security.NewPeerKeys(b), &receiver{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client := NewPeerClient(a)
	defer client.Close()
	var replayBody []byte
	var replaySignature string
	mode := "valid"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		body := recorder.Body.Bytes()
		signature := recorder.Header().Get("X-Clipare-Response-Signature")
		if mode == "valid" {
			replayBody = append([]byte(nil), body...)
			replaySignature = signature
		}
		switch mode {
		case "tampered":
			body = []byte(`{"status":"ok","device":"` + b.Device.ID + `","mode":"bidirectional","platform":"windows"}`)
			signature = "invalid"
		case "unsigned":
			signature = ""
		case "replay":
			body = replayBody
			signature = replaySignature
		case "wrong-id":
			body = []byte(`{"status":"ok","device":"someone-else","platform":"darwin"}`)
		}
		w.Header().Set("X-Clipare-Response-Signature", signature)
		w.WriteHeader(recorder.Code)
		w.Write(body)
	}))
	defer server.Close()
	for _, test := range []string{"valid", "tampered", "unsigned", "replay", "wrong-id", "valid"} {
		mode = test
		h, err := client.HealthPeer(context.Background(), b.Device.ID, server.URL)
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if test == "valid" {
			want = platform(runtime.GOOS)
		}
		if h.Platform != want || client.Platforms()[b.Device.ID] != want {
			t.Fatalf("%s: untrusted platform %q", test, h.Platform)
		}
	}
	// Legacy/global keys never provide device-specific evidence.
	legacy := NewClient(security.StaticSecret(b.Security.Secret))
	defer legacy.Close()
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Health{Status: "ok", Device: b.Device.ID, Platform: "windows"})
	}))
	defer old.Close()
	h, err := legacy.Health(context.Background(), old.URL)
	if err != nil || h.Platform != "" {
		t.Fatal("legacy platform trusted")
	}
}
