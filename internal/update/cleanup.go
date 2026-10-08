package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Cleanup only removes updater-owned, marked directories. A completed helper
// gets a grace period; abandoned downloads expire after seven days. No install
// backups are pruned here (a failed rollback must remain recoverable).
func Cleanup(cache string, now time.Time) {
	entries, e := os.ReadDir(cache)
	if e != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "stage-") {
			continue
		}
		work := filepath.Join(cache, entry.Name())
		b, e := os.ReadFile(filepath.Join(work, ".clipare-stage"))
		var mark struct {
			Created time.Time `json:"created"`
		}
		if e != nil || len(b) > 1024 || json.Unmarshal(b, &mark) != nil || mark.Created.IsZero() {
			continue
		}
		completed, e := os.Stat(filepath.Join(work, "completed"))
		if e == nil && now.Sub(completed.ModTime()) > time.Minute || now.Sub(mark.Created) > 7*24*time.Hour {
			os.RemoveAll(work)
		}
	}
}
