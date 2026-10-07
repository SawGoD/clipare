package app

import (
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"clipare/internal/security"
	clipsync "clipare/internal/sync"
	"clipare/internal/transport"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

var ErrAddressUnavailable = errors.New("Локальный адрес ещё недоступен")

func listenError(err error) error {
	// Winsock returns WSAEADDRNOTAVAIL (10049); Go's Windows
	// syscall.EADDRNOTAVAIL is a synthetic application error, not this value.
	if errors.Is(err, syscall.EADDRNOTAVAIL) || (runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(10049))) {
		return ErrAddressUnavailable
	}
	return errors.New("Не удалось слушать выбранный адрес. Проверьте NetBird, IP и порт; возможно, Clipare уже запущен")
}

type Session struct {
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	err    error
}

func Start(parent context.Context, c config.Config, b clipboard.Backend, log *slog.Logger, health func(string, bool)) (*Session, error) {
	return StartWithControl(parent, c, b, log, health, nil)
}
func StartWithControl(parent context.Context, c config.Config, b clipboard.Backend, log *slog.Logger, health func(string, bool), control http.Handler) (*Session, error) {
	listener, e := net.Listen("tcp", c.ListenAddress())
	if e != nil {
		return nil, listenError(e)
	}
	ctx, cancel := context.WithCancel(parent)
	client := transport.NewPeerClient(c)
	workers := transport.StartWorkersWithStatus(ctx, client, c.Peers, log, health)
	manager := clipsync.NewManager(b, c.Device.ID, c.Mode(), log, workers.Broadcast)
	if e = manager.Initialize(); e != nil {
		cancel()
		listener.Close()
		workers.Wait()
		client.Close()
		return nil, errors.New("Буфер обмена недоступен")
	}
	s := &Session{cancel: cancel, done: make(chan struct{})}
	clipboardHandler := transport.Handler(c, security.NewPeerKeys(c), manager, log)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if control != nil && (r.URL.Path == "/api/v1/discovery" || strings.HasPrefix(r.URL.Path, "/api/v1/pair/") || r.URL.Path == "/api/v1/group/membership") {
			control.ServeHTTP(w, r)
			return
		}
		clipboardHandler.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192, BaseContext: func(net.Listener) context.Context { return ctx }}
	go func() {
		defer close(s.done)
		defer client.Close()
		defer workers.Wait()
		defer cancel()
		results := make(chan error, 2)
		events := make(chan struct{}, 1)
		go func() {
			e := server.Serve(listener)
			if errors.Is(e, http.ErrServerClosed) {
				e = nil
			}
			results <- e
		}()
		go func() { results <- b.Watch(ctx, events) }()
		completed := 0
		var failure error
		log.Info("Clipare started", "device", c.Device.ID, "listen", c.ListenAddress(), "mode", c.Mode())
	running:
		for {
			select {
			case <-ctx.Done():
				break running
			case e := <-results:
				failure = e
				completed++
				break running
			case <-events:
				manager.LocalChanged()
			}
		}
		cancel()
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		if e := server.Shutdown(shutdown); e != nil {
			server.Close()
			if failure == nil {
				failure = errors.New("HTTP shutdown timed out")
			}
		}
		stop()
		for completed < 2 {
			e := <-results
			completed++
			if failure == nil {
				failure = e
			}
		}
		s.mu.Lock()
		s.err = failure
		s.mu.Unlock()
		log.Info("Clipare stopped")
	}()
	return s, nil
}
func (s *Session) Stop() error           { s.cancel(); <-s.done; s.mu.Lock(); defer s.mu.Unlock(); return s.err }
func (s *Session) Done() <-chan struct{} { return s.done }
func (s *Session) Err() error            { s.mu.Lock(); defer s.mu.Unlock(); return s.err }
