package update

import (
	"errors"
	"strconv"
	"strings"
)

// StableVersion deliberately rejects pre-release/dev builds: they never install
// stable updates automatically. Build metadata does not affect precedence.
func StableVersion(s string) ([3]uint64, error) {
	var v [3]uint64
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, "+", 2)
	if len(parts) == 2 {
		for _, p := range strings.Split(parts[1], ".") {
			if !identifier(p) {
				return v, errors.New("invalid version")
			}
		}
	}
	fields := strings.Split(parts[0], ".")
	if len(fields) != 3 {
		return v, errors.New("not a stable version")
	}
	for i, p := range fields {
		if p == "" || len(p) > 1 && p[0] == '0' {
			return v, errors.New("invalid version")
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return v, errors.New("not a stable version")
			}
		}
		n, e := strconv.ParseUint(p, 10, 64)
		if e != nil {
			return v, e
		}
		v[i] = n
	}
	return v, nil
}
func identifier(p string) bool {
	if p == "" {
		return false
	}
	for _, c := range p {
		if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '-') {
			return false
		}
	}
	return true
}
func Newer(latest, current string) bool {
	a, e := StableVersion(latest)
	if e != nil {
		return false
	}
	b, e := StableVersion(current)
	if e != nil {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}
