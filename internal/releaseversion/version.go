// Package releaseversion validates safe build/tag versions. It deliberately
// does not change the updater's stable-only selection or installation policy.
package releaseversion

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var pattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

type Version struct {
	Core       string
	Prerelease bool
}

func Parse(s string) (Version, error) {
	if !pattern.MatchString(s) {
		return Version{}, errors.New("release version must be MAJOR.MINOR.PATCH[-PRERELEASE]")
	}
	core, suffix, pre := strings.Cut(s, "-")
	for _, p := range strings.Split(core, ".") {
		if _, err := strconv.ParseUint(p, 10, 64); err != nil {
			return Version{}, errors.New("release version number is out of range")
		}
	}
	if pre {
		for _, p := range strings.Split(suffix, ".") {
			numeric := true
			for _, c := range p {
				if c < '0' || c > '9' {
					numeric = false
					break
				}
			}
			if numeric && len(p) > 1 && p[0] == '0' {
				return Version{}, errors.New("numeric prerelease identifier has a leading zero")
			}
		}
	}
	return Version{core, pre}, nil
}
