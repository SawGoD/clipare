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
	"log/slog"
	"net/http"
	"time"
)

type Receiver interface {
	Receive(context.Context, clipsync.Message) error
}

func Handler(c config.Config, secret security.SecretProvider, receiver Receiver, log *slog.Logger) http.Handler {
	trusted := map[string]bool{}
	for _, p := range c.Peers {
		trusted[p.ID] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" && r.URL.Path != "/api/v1/clipboard" {
			http.NotFound(w, r)
			return
		}
		method := "POST"
		if r.URL.Path == "/api/v1/health" {
			method = "GET"
		}
		if r.Method != method {
			w.Header().Set("Allow", method)
			http.Error(w, "method not allowed", 405)
			return
		}
		body, e := io.ReadAll(http.MaxBytesReader(w, r.Body, clipsync.MaxBody))
		if e != nil {
			http.Error(w, "request too large", 413)
			return
		}
		if !security.Verify(secret.Secret(), r.Header.Get(security.TimestampHeader), r.Header.Get(security.SignatureHeader), body, time.Now()) {
			log.Warn("invalid request signature")
			http.Error(w, "unauthorized", 401)
			return
		}
		if method == "GET" {
			if len(body) != 0 {
				http.Error(w, "unexpected body", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Health{"ok", c.Device.ID, c.Mode()})
			return
		}
		var msg clipsync.Message
		d := json.NewDecoder(bytes.NewReader(body))
		d.DisallowUnknownFields()
		if d.Decode(&msg) != nil {
			http.Error(w, "invalid message", 400)
			return
		}
		var extra any
		if d.Decode(&extra) != io.EOF || msg.Validate() != nil {
			http.Error(w, "invalid message", 400)
			return
		}
		if !trusted[msg.Source] {
			http.Error(w, "untrusted source", 403)
			return
		}
		now := time.Now().Unix()
		if msg.Timestamp < now-60 || msg.Timestamp > now+60 {
			http.Error(w, "stale message", 400)
			return
		}
		if e := receiver.Receive(r.Context(), msg); e != nil {
			if errors.Is(e, clipsync.ErrDisabled) {
				http.Error(w, "sync disabled", 409)
			} else {
				http.Error(w, "clipboard unavailable", 503)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
