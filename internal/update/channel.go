package update

import (
	"clipare/internal/releaseversion"
	"errors"
	"strings"
)

const (
	ChannelStable = "stable"
	ChannelBeta   = "beta"
)

// An unset preference follows the installed release; explicit choices persist.
func ResolveChannel(preference, current string) string {
	if preference == ChannelStable || preference == ChannelBeta {
		return preference
	}
	if v, err := parseVersion(current); err == nil && v.prerelease != "" {
		return ChannelBeta
	}
	return ChannelStable
}

type semanticVersion struct {
	core       [3]uint64
	prerelease string
}

func parseVersion(raw string) (semanticVersion, error) {
	var v semanticVersion
	s := strings.TrimPrefix(raw, "v")
	version, metadata, hasMetadata := strings.Cut(s, "+")
	if hasMetadata {
		for _, p := range strings.Split(metadata, ".") {
			if !identifier(p) {
				return v, errors.New("invalid version metadata")
			}
		}
	}
	parsed, err := releaseversion.Parse(version)
	if err != nil {
		return v, err
	}
	v.core, err = StableVersion(parsed.Core)
	if parsed.Prerelease {
		_, v.prerelease, _ = strings.Cut(version, "-")
	}
	return v, err
}

func compareVersions(a, b semanticVersion) int {
	for i := range a.core {
		if a.core[i] > b.core[i] {
			return 1
		}
		if a.core[i] < b.core[i] {
			return -1
		}
	}
	if a.prerelease == b.prerelease {
		return 0
	}
	if a.prerelease == "" {
		return 1
	}
	if b.prerelease == "" {
		return -1
	}
	x, y := strings.Split(a.prerelease, "."), strings.Split(b.prerelease, ".")
	numeric := func(s string) bool {
		for _, r := range s {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] == y[i] {
			continue
		}
		xn, yn := numeric(x[i]), numeric(y[i])
		if xn != yn {
			if xn {
				return -1
			}
			return 1
		}
		if xn && len(x[i]) != len(y[i]) {
			if len(x[i]) > len(y[i]) {
				return 1
			}
			return -1
		}
		if x[i] > y[i] {
			return 1
		}
		return -1
	}
	if len(x) > len(y) {
		return 1
	}
	return -1
}

// Development builds are never installation candidates, even on beta.
func DevelopmentVersion(s string) bool {
	v, err := parseVersion(s)
	if err != nil {
		return true
	}
	for _, p := range strings.Split(strings.ToLower(v.prerelease), ".") {
		if p == "dev" || p == "snapshot" {
			return true
		}
	}
	return false
}

func packageVersion(s string) (releaseversion.Version, error) {
	if DevelopmentVersion(s) {
		return releaseversion.Version{}, errors.New("development package")
	}
	return releaseversion.Parse(s)
}

func DisplayVersion(s string) string {
	if v, err := parseVersion(s); err == nil && v.prerelease != "" && !DevelopmentVersion(s) {
		return s + " (бета)"
	}
	return s
}
