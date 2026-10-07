package transport

import (
	"bytes"
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
}

func NewClient(secret security.SecretProvider) *Client {
	return &Client{http: &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, MaxIdleConns: 128, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, secret: secret}
}
func (c *Client) Close() { c.http.CloseIdleConnections() }
func (c *Client) request(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	r, e := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if e != nil {
		return nil, errors.New("invalid peer request")
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r.Header.Set(security.TimestampHeader, ts)
	r.Header.Set(security.SignatureHeader, security.Sign(c.secret.Secret(), ts, body))
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
	b, e := json.Marshal(m)
	if e != nil {
		return e
	}
	_, e = c.request(ctx, "POST", url+"/api/v1/clipboard", b)
	return e
}

type Health struct {
	Status string `json:"status"`
	Device string `json:"device"`
	Mode   string `json:"mode"`
}

func (c *Client) Health(ctx context.Context, url string) (Health, error) {
	b, e := c.request(ctx, "GET", url+"/api/v1/health", nil)
	if e != nil {
		return Health{}, e
	}
	var h Health
	if json.Unmarshal(b, &h) != nil || h.Status != "ok" {
		return h, errors.New("invalid health response")
	}
	return h, nil
}
