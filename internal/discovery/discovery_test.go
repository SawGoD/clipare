package discovery

import (
	"context"
	"errors"
	"testing"
)

const fixture = `{"peers":{"details":[{"fqdn":"desktop.netbird.cloud","netbirdIp":"100.64.0.2","status":"Connected"},{"fqdn":"offline","netbirdIp":"100.64.0.3","status":"Disconnected"},{"netbirdIp":"8.8.8.8","status":"Connected"}]}}`

type fakeRunner struct {
	b   []byte
	err error
}

func (r fakeRunner) Status(context.Context) ([]byte, error) { return r.b, r.err }

type fakeProbe struct{ err error }

func (p fakeProbe) Probe(ctx context.Context, peer Peer) (Info, error) {
	return Info{DeviceID: "pc", DeviceName: "Desktop", PairingAvailable: true}, p.err
}
func TestParseAndScan(t *testing.T) {
	p, err := Parse([]byte(fixture))
	if err != nil || len(p) != 1 || p[0].Name != "desktop" {
		t.Fatal("JSON parsing")
	}
	for _, b := range []string{"bad", "{}", "null"} {
		if _, err = Parse([]byte(b)); err == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
	r := fakeRunner{[]byte(fixture), nil}
	got, err := Scan(context.Background(), r, fakeProbe{}, nil)
	if err != nil || len(got) != 1 {
		t.Fatal("scan")
	}
	got, err = Scan(context.Background(), r, fakeProbe{}, map[string]bool{"pc": true})
	if err != nil || len(got) != 0 {
		t.Fatal("already paired")
	}
	got, err = Scan(context.Background(), r, fakeProbe{errors.New("absent")}, nil)
	if err != nil || len(got) != 0 {
		t.Fatal("absent endpoint")
	}
	if _, err = Scan(context.Background(), fakeRunner{err: errors.New("missing binary")}, fakeProbe{}, nil); !errors.Is(err, ErrUnavailable) {
		t.Fatal("CLI fallback")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = Scan(ctx, r, fakeProbe{}, nil); err == nil {
		t.Fatal("cancellation")
	}
}
