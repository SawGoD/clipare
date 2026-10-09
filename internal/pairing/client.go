package pairing

import (
	"bytes"
	"clipare/internal/discovery"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

var ErrPeerUnavailable = errors.New("Устройство больше недоступно")

func pairClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, MaxIdleConns: 4, IdleConnTimeout: 15 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
func post(ctx context.Context, client *http.Client, url, path string, v, out any) error {
	b, _ := json.Marshal(v)
	r, e := http.NewRequestWithContext(ctx, "POST", url+path, bytes.NewReader(b))
	if e != nil {
		return ErrInvalid
	}
	r.Header.Set("Content-Type", "application/json")
	res, e := client.Do(r)
	if e != nil {
		if ctx.Err() != nil {
			return contextError(ctx)
		}
		return ErrPeerUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode == 410 {
		return ErrExpired
	}
	if res.StatusCode == 412 {
		return ErrGroupMerge
	}
	if res.StatusCode == 422 {
		return ErrIdentityChanged
	}
	if res.StatusCode == 423 {
		return ErrRemoved
	}
	if res.StatusCode == 409 {
		return errors.New("Другое подключение уже выполняется или pairing приостановлен. Завершите подключение и включите синхронизацию на другом устройстве")
	}
	if res.StatusCode == 429 {
		return errors.New("Слишком много запросов подключения. Подождите несколько секунд и повторите")
	}
	if res.StatusCode != 200 {
		return errors.New("Не удалось подключиться. Попробуйте ещё раз")
	}
	b, e = io.ReadAll(io.LimitReader(res.Body, (128<<10)+1))
	if e != nil || len(b) > 128<<10 {
		return ErrInvalid
	}
	return decode(b, out)
}

// Connect reports SAS only after commitment and key confirmation succeeded.
// It persists trusted metadata only after the remote user's explicit approval.
func (s *Service) Connect(ctx context.Context, d discovery.Device, show func(string), verify ...func(context.Context) error) error {
	s.pairGate.Lock()
	if s.disabled.Load() || !s.sessions.Available() || !s.outgoing.CompareAndSwap(false, true) {
		s.pairGate.Unlock()
		return errors.New("Другое подключение уже выполняется")
	}
	s.pairGate.Unlock()
	defer s.outgoing.Store(false)
	if !discovery.NetBirdAddress(d.IP) {
		return ErrInvalid
	}
	c := s.Config()
	if len(c.Peers) > 0 && len(verify) == 0 {
		return errors.New("Подтвердите совпадение кода на существующем устройстве")
	}
	for _, p := range c.Peers {
		if p.ID == d.DeviceID {
			return errors.New("Это устройство уже добавлено")
		}
	}
	h, e := NewHandshake(c)
	if e != nil {
		return e
	}
	id, e := NewID()
	if e != nil {
		return e
	}
	client := pairClient()
	if s.client != nil {
		client = s.client
	}
	defer client.CloseIdleConnections()
	url := "http://" + net.JoinHostPort(d.IP, strconv.Itoa(discovery.Port))
	var ch Challenge
	if e = post(ctx, client, url, "/api/v1/pair/request", Request{ExistingGroup: h.Hello.ExistingGroup, Session: id, Commitment: Commitment(h.Hello), Expires: h.Hello.Expires}, &ch); e != nil {
		return e
	}
	if ch.Session != id || ch.Hello.ID != d.DeviceID || ch.Hello.PublicKey != d.PublicKey || ch.Hello.IP != d.IP || ch.Expires <= time.Now().Unix() || ch.Expires > time.Now().Add(180*time.Second).Unix() {
		return ErrInvalid
	}
	if len(c.Peers) > 0 && ch.Group != c.Group.ID {
		return ErrGroupMerge
	}
	ctx, stop := context.WithDeadline(ctx, time.Unix(ch.Expires, 0))
	defer stop()
	key, sas, e := h.Keys(ch.Hello, id, ch.Group, ch.Expires, true)
	if e != nil {
		return e
	}
	// The server limits requests to one per second per source.
	if e = wait(ctx, time.Second); e != nil {
		return e
	}
	var confirmation Confirmation
	if e = post(ctx, client, url, "/api/v1/pair/exchange", Exchange{Session: id, Hello: h.Hello, Proof: proof(key, id, "reveal")}, &confirmation); e != nil {
		return e
	}
	if confirmation.Session != id || !verifyProof(key, id, "exchange", confirmation.Proof) {
		return ErrInvalid
	}
	show(sas)
	finished := false
	defer func() {
		if finished {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = wait(cleanup, time.Second)
		var out struct {
			Status string `json:"status"`
		}
		_ = post(cleanup, client, url, "/api/v1/pair/cancel", Confirmation{id, proof(key, id, "cancel")}, &out)
	}()
	if len(c.Peers) > 0 {
		if e = verify[0](ctx); e != nil {
			if errors.Is(e, context.DeadlineExceeded) {
				return ErrExpired
			}
			return e
		}
	}
	if e = wait(ctx, time.Second); e != nil {
		return e
	}
	var confirmed Confirmation
	if e = post(ctx, client, url, "/api/v1/pair/confirm", Confirmation{id, proof(key, id, "confirm")}, &confirmed); e != nil {
		return e
	}
	if confirmed.Session != id || !verifyProof(key, id, "confirm-response", confirmed.Proof) {
		return ErrInvalid
	}
	failures := 0
	for time.Now().Unix() < ch.Expires {
		if e = wait(ctx, 1500*time.Millisecond); e != nil {
			return e
		}
		var state Status
		if e = post(ctx, client, url, "/api/v1/pair/status", Confirmation{id, proof(key, id, "status")}, &state); e != nil {
			if errors.Is(e, ErrPeerUnavailable) && failures < 3 {
				failures++
				continue
			}
			return e
		}
		failures = 0
		if !verifyProof(key, id, statusAction(state), state.Proof) {
			return ErrInvalid
		}
		switch state.State {
		case Rejected:
			return errors.New("Устройство отклонило подключение")
		case Approved:
			if state.Membership.Group != ch.Group {
				return ErrInvalid
			}
			for _, m := range state.Membership.Members {
				if m.ID == ch.Hello.ID && m.PublicKey != ch.Hello.PublicKey {
					return ErrInvalid
				}
			}
			if e = s.adopt(state.Membership, ch.Hello.ID, true, d.FQDN); e != nil {
				return e
			}
			finished = true
			if e = wait(ctx, time.Second); e != nil {
				return e
			}
			var done struct {
				Status string `json:"status"`
			}
			_ = post(ctx, client, url, "/api/v1/pair/finish", Confirmation{id, proof(key, id, "finish")}, &done)
			return nil // Both sides have persisted trust; finish only erases ephemeral state.
		}
	}
	return ErrExpired
}
func wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return contextError(ctx)
	}
}
func contextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ErrExpired
	}
	return ctx.Err()
}
