package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("Автоматическое обнаружение недоступно. Проверьте, что NetBird запущен")

type Peer struct {
	Name string
	IP   string
	FQDN string
}
type Runner interface {
	Status(context.Context) ([]byte, error)
}
type CLI struct{}
type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 1<<20 {
		return 0, errors.New("NetBird status too large")
	}
	return b.Buffer.Write(p)
}
func (CLI) Status(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	binary, err := findBinary(exec.LookPath, func(path string) bool {
		st, err := os.Stat(path)
		return err == nil && st.Mode().IsRegular() && (runtime.GOOS == "windows" || st.Mode().Perm()&0111 != 0)
	}, standardPaths())
	if err != nil {
		return nil, ErrUnavailable
	}
	cmd := exec.CommandContext(ctx, binary, "status", "--json")
	var b limitedBuffer
	cmd.Stdout = &b
	cmd.Stderr = io.Discard
	if cmd.Run() != nil {
		return nil, ErrUnavailable
	}
	return b.Bytes(), nil
}
func standardPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/usr/local/bin/netbird", "/opt/homebrew/bin/netbird"}
	case "windows":
		return []string{filepath.Join(os.Getenv("ProgramFiles"), "Netbird", "netbird.exe")}
	}
	return nil
}
func findBinary(lookup func(string) (string, error), exists func(string) bool, candidates []string) (string, error) {
	if p, err := lookup("netbird"); err == nil {
		return p, nil
	}
	for _, p := range candidates {
		if filepath.IsAbs(p) && exists(p) {
			return p, nil
		}
	}
	return "", ErrUnavailable
}

// NetBirdAddress deliberately excludes public, LAN and routed subnet addresses.
// Only the IPv4 CGNAT range assigned to NetBird hosts is probed in this version.
func NetBirdAddress(s string) bool {
	a, err := netip.ParseAddr(s)
	return err == nil && netip.MustParsePrefix("100.64.0.0/10").Contains(a)
}
func Parse(b []byte) ([]Peer, error) {
	if len(b) > 1<<20 {
		return nil, ErrUnavailable
	}
	var status struct {
		DaemonStatus string `json:"daemonStatus"`
		Peers        *struct {
			Details []struct {
				FQDN   string `json:"fqdn"`
				IP     string `json:"netbirdIp"`
				Status string `json:"status"`
				Name   string `json:"hostname"`
			} `json:"details"`
		} `json:"peers"`
	}
	if json.Unmarshal(b, &status) != nil || status.Peers == nil || len(status.Peers.Details) > 1024 {
		return nil, ErrUnavailable
	}
	if status.DaemonStatus != "" && !strings.EqualFold(status.DaemonStatus, "Connected") {
		return nil, ErrUnavailable
	}
	var peers []Peer
	seen := map[string]bool{}
	for _, p := range status.Peers.Details {
		if !strings.EqualFold(p.Status, "Connected") || !NetBirdAddress(p.IP) || seen[p.IP] {
			continue
		}
		seen[p.IP] = true
		name := p.Name
		if name == "" {
			name = strings.Split(p.FQDN, ".")[0]
		}
		if name == "" {
			name = p.IP
		}
		peers = append(peers, Peer{name, p.IP, p.FQDN})
	}
	return peers, nil
}

// LocalAddress accepts the CLI's CIDR-form local address without confusing it
// with another CGNAT VPN interface. It is never inferred from a public IP.
func LocalAddress(b []byte) string {
	if _, err := Parse(b); err != nil {
		return ""
	}
	var v struct {
		IP string `json:"netbirdIp"`
	}
	if json.Unmarshal(b, &v) != nil {
		return ""
	}
	address := strings.Split(v.IP, "/")[0]
	if NetBirdAddress(address) {
		return address
	}
	return ""
}
