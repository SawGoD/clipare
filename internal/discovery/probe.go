package discovery

import (
	"clipare/internal/identity"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Port = 45873

type Info struct {
	Protocol         int    `json:"protocol"`
	App              string `json:"app"`
	Version          string `json:"version"`
	DeviceID         string `json:"device_id"`
	DeviceName       string `json:"device_name"`
	PublicKey        string `json:"public_key"`
	PairingAvailable bool   `json:"pairing_available"`
}
type Device struct {
	Peer
	Info
}
type Prober interface {
	Probe(context.Context, Peer) (Info, error)
}
type HTTPProber struct{ Client *http.Client }

func NewProber() *HTTPProber {
	return &HTTPProber{&http.Client{Timeout: 750 * time.Millisecond, Transport: &http.Transport{Proxy: nil, MaxIdleConns: 4, IdleConnTimeout: 15 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (p *HTTPProber) Close() { p.Client.CloseIdleConnections() }
func (p *HTTPProber) Probe(ctx context.Context, peer Peer) (Info, error) {
	if !NetBirdAddress(peer.IP) {
		return Info{}, ErrUnavailable
	}
	r, err := http.NewRequestWithContext(ctx, "GET", "http://"+net.JoinHostPort(peer.IP, strconv.Itoa(Port))+"/api/v1/discovery", nil)
	if err != nil {
		return Info{}, err
	}
	res, err := p.Client.Do(r)
	if err != nil {
		return Info{}, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4097))
	if err != nil || len(b) > 4096 || res.StatusCode != 200 {
		return Info{}, ErrUnavailable
	}
	var i Info
	if json.Unmarshal(b, &i) != nil || i.Protocol != 1 || i.App != "clipare" || i.DeviceID == "" || len(i.DeviceID) > 128 || len(i.DeviceName) > 256 || strings.ContainsAny(i.DeviceName, "\n\r\x00") || len(i.Version) > 64 {
		return Info{}, ErrUnavailable
	}
	if _, err = identity.Public(i.PublicKey); err != nil {
		return Info{}, ErrUnavailable
	}
	return i, nil
}

// Scan owns a fixed worker pool. Cancellation joins every worker before return.
func Scan(ctx context.Context, runner Runner, prober Prober, known map[string]bool) ([]Device, error) {
	b, err := runner.Status(ctx)
	if err != nil {
		return nil, ErrUnavailable
	}
	peers, err := Parse(b)
	if err != nil {
		return nil, err
	}
	jobs := make(chan Peer)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var found []Device
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				probeCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
				info, e := prober.Probe(probeCtx, p)
				cancel()
				if e == nil && !known[info.DeviceID] && info.PairingAvailable {
					mu.Lock()
					found = append(found, Device{p, info})
					mu.Unlock()
				}
			}
		}()
	}
loop:
	for _, p := range peers {
		select {
		case jobs <- p:
		case <-ctx.Done():
			break loop
		}
	}
	close(jobs)
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	sort.Slice(found, func(i, j int) bool { return found[i].DeviceName < found[j].DeviceName })
	return found, nil
}
