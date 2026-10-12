package history

import (
	"time"
)

// survivorsToDrop returns the record references to delete: everything past
// the retention window plus the oldest rows beyond the item limit. Locked
// records (unreadable key) are never dropped by cleanup; only an explicit
// clear may remove them.
func survivorsToDrop(refs []recordRef, now time.Time, retention time.Duration, max int) map[recordRef]bool {
	drop := map[recordRef]bool{}
	kept := 0
	for _, ref := range refs {
		if ref.locked {
			continue
		}
		if now.Sub(ref.created) > retention {
			drop[ref] = true
			continue
		}
		kept++
	}
	if kept <= max {
		return drop
	}
	// Newest rows survive; order by created descending, id-independent.
	live := make([]recordRef, 0, kept)
	for _, ref := range refs {
		if !ref.locked && !drop[ref] {
			live = append(live, ref)
		}
	}
	for i := 1; i < len(live); i++ {
		for j := i; j > 0 && live[j].created.After(live[j-1].created); j-- {
			live[j], live[j-1] = live[j-1], live[j]
		}
	}
	for _, ref := range live[max:] {
		drop[ref] = true
	}
	return drop
}
