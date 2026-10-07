package sync

import (
	"clipare/internal/clipboard"
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var ErrDisabled = errors.New("receiving disabled")

type Manager struct {
	mu           sync.Mutex
	backend      clipboard.Backend
	device, mode string
	cache        *Cache
	currentHash  string
	hasCurrent   bool
	latestTime   int64
	latestID     string
	log          *slog.Logger
	broadcast    func(Message)
}

func NewManager(b clipboard.Backend, device, mode string, log *slog.Logger, broadcast func(Message)) *Manager {
	return &Manager{backend: b, device: device, mode: mode, cache: NewCache(1000, 5*time.Minute), log: log, broadcast: broadcast}
}
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok, e := m.backend.Read()
	if errors.Is(e, clipboard.ErrTooLarge) {
		m.log.Warn("initial clipboard exceeds limit")
		return nil
	}
	if e != nil {
		return e
	}
	if ok {
		m.currentHash = Hash(s)
		m.hasCurrent = true
	}
	return nil
}
func (m *Manager) LocalChanged() {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok, e := m.backend.Read()
	if e != nil {
		if errors.Is(e, clipboard.ErrTooLarge) {
			m.log.Warn("clipboard text exceeds limit")
		} else {
			m.log.Warn("clipboard read failed")
		}
		return
	}
	if !ok {
		m.hasCurrent = false
		return
	}
	h := Hash(s)
	if m.hasCurrent && h == m.currentHash {
		return
	}
	m.currentHash = h
	m.hasCurrent = true
	if m.mode != "bidirectional" && m.mode != "send-only" {
		return
	}
	msg, e := newMessageAfter(m.device, s, m.latestTime, m.latestID)
	if e != nil || msg.Validate() != nil {
		m.log.Warn("clipboard text rejected", "size", len(s))
		return
	}
	m.cache.Add(msg.ID, time.Now())
	m.latestTime = msg.Timestamp
	m.latestID = msg.ID
	m.log.Info("clipboard changed", "id", msg.ID, "hash", h, "size", len(s))
	m.broadcast(msg)
}
func (m *Manager) Receive(ctx context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := msg.Validate(); e != nil {
		return e
	}
	if m.mode != "bidirectional" && m.mode != "receive-only" {
		return ErrDisabled
	}
	now := time.Now()
	if m.cache.Seen(msg.ID, now) || msg.Source == m.device {
		return nil
	}
	// Total order resolves simultaneous changes consistently across fully meshed peers.
	if msg.Timestamp < m.latestTime || (msg.Timestamp == m.latestTime && msg.ID <= m.latestID) {
		m.cache.Add(msg.ID, now)
		return nil
	}
	h := Hash(msg.Data)
	// The OS clipboard may have changed before its pending notification is read.
	// Apply every new winning ID rather than trusting our last observed hash.
	if e := m.backend.Write(msg.Data); e != nil {
		m.log.Error("clipboard write failed", "id", msg.ID)
		return errors.New("clipboard write failed")
	}
	m.currentHash = h
	m.hasCurrent = true
	m.latestTime = msg.Timestamp
	m.latestID = msg.ID
	m.cache.Add(msg.ID, now)
	m.log.Info("clipboard received", "source", msg.Source, "id", msg.ID, "hash", h, "size", len(msg.Data))
	return nil
}
