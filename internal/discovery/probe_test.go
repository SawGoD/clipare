package discovery

import (
	"clipare/internal/identity"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type responseTransport struct {
	body   string
	status int
}

func (t responseTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: t.status, Body: io.NopCloser(strings.NewReader(t.body)), Header: make(http.Header)}, nil
}
func TestHTTPProbeValidation(t *testing.T) {
	id, _ := identity.Generate()
	info := Info{Protocol: 1, App: "clipare", Version: "0.3.0", DeviceID: "pc", DeviceName: "Desktop", PublicKey: id.PublicKey, PairingAvailable: true}
	b, _ := json.Marshal(info)
	p := &HTTPProber{Client: &http.Client{Transport: responseTransport{string(b), 200}}}
	if got, err := p.Probe(context.Background(), Peer{IP: "100.64.0.2"}); err != nil || got.DeviceID != "pc" {
		t.Fatal("valid probe")
	}
	for _, tc := range []responseTransport{{"{}", 200}, {string(b), 404}, {strings.Repeat("x", 4097), 200}} {
		p.Client.Transport = tc
		if _, err := p.Probe(context.Background(), Peer{IP: "100.64.0.2"}); err == nil {
			t.Fatal("invalid probe accepted")
		}
	}
	if _, err := p.Probe(context.Background(), Peer{IP: "8.8.8.8"}); err == nil {
		t.Fatal("public IP probe")
	}
}

type timeoutProbe struct{}

func (timeoutProbe) Probe(ctx context.Context, p Peer) (Info, error) {
	<-ctx.Done()
	return Info{}, ctx.Err()
}
func TestScanTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := Scan(ctx, fakeRunner{b: []byte(fixture)}, timeoutProbe{}, nil); err == nil {
		t.Fatal("scan did not time out")
	}
}
func TestPublicEndpointAndRateLimit(t *testing.T) {
	h := Handler(func() Info { return Info{Protocol: 1, App: "clipare", DeviceID: "pc", DeviceName: "Desktop"} })
	r := httptest.NewRequest("GET", "/api/v1/discovery", nil)
	r.RemoteAddr = "100.64.0.2:1"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal("public metadata")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 429 {
		t.Fatal("probe spam not limited")
	}
}
