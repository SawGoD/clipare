package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const Interval = 24 * time.Hour

type State struct {
	LastCheck time.Time `json:"last_check"`
}
type Checker struct {
	GitHub    GitHub
	Current   string
	StatePath string
}

func (c Checker) Due(now time.Time) bool {
	b, e := os.ReadFile(c.StatePath)
	var s State
	if e != nil || json.Unmarshal(b, &s) != nil {
		return true
	}
	return s.LastCheck.IsZero() || now.Before(s.LastCheck) || now.Sub(s.LastCheck) >= Interval
}

// Check records attempts (including failures) so restarts cannot hammer GitHub.
// Manual requests bypass the interval, never the user-confirmation requirement.
func (c Checker) Check(ctx context.Context, enabled, manual bool) (*Release, error) {
	if _, e := StableVersion(c.Current); e != nil {
		return nil, nil
	}
	if !manual && (!enabled || !c.Due(time.Now())) {
		return nil, nil
	}
	if e := os.MkdirAll(filepath.Dir(c.StatePath), 0700); e != nil {
		return nil, e
	}
	b, _ := json.Marshal(State{time.Now().UTC()})
	if e := atomicWrite(c.StatePath, b); e != nil {
		return nil, e
	}
	r, e := c.GitHub.Latest(ctx)
	if e != nil {
		return nil, e
	}
	if !Newer(r.Version, c.Current) {
		return nil, nil
	}
	return &r, nil
}
func CacheDir() (string, error) {
	p, e := os.UserCacheDir()
	return filepath.Join(p, "Clipare", "update"), e
}
func atomicWrite(path string, b []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".state-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return replaceState(f.Name(), path)
}
