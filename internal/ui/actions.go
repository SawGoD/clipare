package ui

import "strings"

// Shared input requirements; native controls and the event loop use the same
// rules. Whitespace-only input never enables an action.
func actionReady(action int, f form) bool {
	required := []int{}
	switch action {
	case eventSave:
		required = []int{0, 2, 3, 4}
	case eventDeviceName:
		required = []int{0}
	case eventImport:
		required = []int{10}
	case eventUpsert:
		required = []int{6, 7, 8, 9}
	case eventCopy:
		required = []int{0, 2, 3, 4}
	}
	for _, i := range required {
		if strings.TrimSpace(f.Values[i]) == "" {
			return false
		}
	}
	return true
}

type actionDesktop interface{ Actions(uint64) }

func actionMask(f form) uint64 {
	var mask uint64
	for _, id := range []int{eventSave, eventImport, eventUpsert, eventCopy, eventDeviceName} {
		if actionReady(id, f) {
			mask |= 1 << id
		}
	}
	return mask
}
func renderActions(d desktop, f form) {
	if n, ok := d.(actionDesktop); ok {
		n.Actions(actionMask(f))
	}
}
