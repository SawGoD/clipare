package transport

import (
	"bytes"
	"clipare/internal/config"
	"clipare/internal/security"
	clipsync "clipare/internal/sync"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	http   *http.Client
	secret security.SecretProvider
	source string
}

func NewPeerClient(c config.Config) *Client {
	client := NewClient(security.NewPeerKeys(c))
	client.source = c.Device.ID
	return client
}

func NewClient(secret security.SecretProvider) *Client {
	return &Client{http: &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, MaxIdleConns: 128, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, secret: secret}
}
func (c *Client) Close() { c.http.CloseIdleConnections() }
func (c *Client) request(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	return c.peerRequest(ctx, "", method, url, body)
}
func (c *Client) peerRequest(ctx context.Context, peerID, method, url string, body []byte) ([]byte, error) {
	r, e := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if e != nil {
		return nil, errors.New("invalid peer request")
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r.Header.Set(security.TimestampHeader, ts)
	key := c.secret.Secret()
	signed := body
	if keys, ok := c.secret.(*security.PeerKeys); ok && peerID != "" {
		key, e = keys.KeyForPeer(peerID)
		if e != nil {
			return nil, e
		}
		if !keys.Legacy(peerID) {
			r.Header.Set(security.SourceHeader, c.source)
			signed = security.AuthenticatedData(method, r.URL.Path, c.source, body)
		}
	}
	r.Header.Set(security.SignatureHeader, security.Sign(key, ts, signed))
	r.Header.Set("Content-Type", "application/json")
	res, e := c.http.Do(r)
	if e != nil {
		return nil, errors.New("peer request failed")
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, 4097))
	if e != nil || len(b) > 4096 {
		return nil, errors.New("invalid peer response")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("peer rejected request")
	}
	return b, nil
}
func (c *Client) Send(ctx context.Context, url string, m clipsync.Message) error {
	return c.SendPeer(ctx, "", url, m)
}
func (c *Client) SendPeer(ctx context.Context, peerID, url string, m clipsync.Message) error {
	b, e := json.Marshal(m)
	if e != nil {
		return e
	}
	_, e = c.peerRequest(ctx, peerID, "POST", url+"/api/v1/clipboard", b)
	return e
}

type Health struct {
	Status string `json:"status"`
	Device string `json:"device"`
	Mode   string `json:"mode"`
}

func (c *Client) Health(ctx context.Context, url string) (Health, error) {
	return c.HealthPeer(ctx, "", url)
}
func (c *Client) HealthPeer(ctx context.Context, peerID, url string) (Health, error) {
	b, e := c.peerRequest(ctx, peerID, "GET", url+"/api/v1/health", nil)
	if e != nil {
		return Health{}, e
	}
	var h Health
	if json.Unmarshal(b, &h) != nil || h.Status != "ok" {
		return h, errors.New("invalid health response")
	}
	return h, nil
}
