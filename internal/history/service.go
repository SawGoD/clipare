package history

import (
	"clipare/internal/config"
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoryFile is the encrypted history database file name, kept next to the
// user configuration.
const HistoryFile = "history.db"

// Service records clipboard events into the local encrypted store and serves
// the History UI. It implements the app.HistoryRecorder contract.
type Service struct {
	mu        sync.Mutex
	store     *Store
	log       *slog.Logger
	enabled   bool
	retention time.Duration
	maxItems  int
	ownID     string
	ownName   string
	peerNames map[string]string
	keyLost   bool
}

// Open opens the history service with the platform keystore. A missing key is
// generated and stored; an unavailable keystore disables history.
func Open(path string, c config.Config, log *slog.Logger) (*Service, error) {
	ks, e := DefaultKeystore()
	if e != nil {
		return nil, e
	}
	return OpenWithKeystore(path, ks, c, log)
}

// OpenWithKeystore is the injectable constructor used by tests.
func OpenWithKeystore(path string, ks Keystore, c config.Config, log *slog.Logger) (*Service, error) {
	key, e := ks.Load()
	switch {
	case errors.Is(e, os.ErrNotExist):
		key = make([]byte, keySize)
		if _, e = rand.Read(key); e != nil {
			return nil, e
		}
		if e = ks.Save(key); e != nil {
			return nil, errors.New("cannot protect the history key")
		}
	case e != nil:
		return nil, errors.New("history key unavailable")
	}
	store, e := OpenStore(path, key)
	if e != nil {
		return nil, e
	}
	s := &Service{store: store, log: log, peerNames: map[string]string{}}
	s.UpdateSettings(c)
	if e = store.Cleanup(time.Now(), s.retention, s.maxItems); e != nil {
		log.Warn("history cleanup failed")
	}
	return s, nil
}

// UpdateSettings applies the current configuration, including peer display
// names for origin resolution.
func (s *Service) UpdateSettings(c config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = c.History.Enabled
	s.retention = c.HistoryRetention()
	s.maxItems = c.HistoryMaxItems()
	s.ownID = c.Device.ID
	s.ownName = c.Device.Name
	names := make(map[string]string, len(c.Peers))
	for _, p := range c.Peers {
		name := p.Name
		if name == "" {
			name = p.ID
		}
		names[p.ID] = name
	}
	s.peerNames = names
}

// Record stores one clipboard event. The same message never creates a second
// row; identical content refreshes the existing row instead.
func (s *Service) Record(msgID, source, text string, timestamp int64, local bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil || !s.enabled || s.keyLost {
		return
	}
	kind := Classify(text)
	item := Item{
		ID: msgID, OriginID: source, Timestamp: timestamp, Type: kind,
		Text: text, Sensitive: kind == ContentText && LooksSensitive(text),
	}
	if local {
		item.OriginName = s.ownName
	} else if name, ok := s.peerNames[source]; ok {
		item.OriginName = name
	} else {
		item.OriginName = source
	}
	now := time.Now()
	added, e := s.store.Add(item, now)
	if e != nil {
		s.log.Warn("history record failed")
		return
	}
	if added {
		if e = s.store.Cleanup(now, s.retention, s.maxItems); e != nil {
			s.log.Warn("history cleanup failed")
		}
	}
}

// Items returns the decrypted rows, newest first.
func (s *Service) Items() ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return nil, errors.New("history is unavailable")
	}
	items, e := s.store.List()
	if errors.Is(e, ErrKeyLost) {
		s.keyLost = true
	}
	return items, e
}

// Delete removes a single row locally.
func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return errors.New("history is unavailable")
	}
	return s.store.Delete(id)
}

// Clear removes every row; only history data is affected.
func (s *Service) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return errors.New("history is unavailable")
	}
	if e := s.store.Clear(); e != nil {
		return e
	}
	s.keyLost = false
	return nil
}

// Run performs periodic cleanup until the context is cancelled. An initial
// pass already happened when the service was opened.
func (s *Service) Run(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			s.mu.Lock()
			store, retention, max := s.store, s.retention, s.maxItems
			s.mu.Unlock()
			if store == nil {
				return
			}
			if e := store.Cleanup(now, retention, max); e != nil {
				s.log.Warn("history cleanup failed")
			}
		}
	}
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return nil
	}
	e := s.store.Close()
	s.store = nil
	return e
}

// DefaultPath returns the history database path next to the configuration.
func DefaultPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), HistoryFile)
}
